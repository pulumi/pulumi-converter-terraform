// Copyright 2026, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package sshharness runs an in-process SSH server for conformance tests that
// exercise remote-exec provisioners. The server listens on an ephemeral
// loopback port, accepts any public key or password, and records every
// "exec" request (plus SCP-uploaded file contents) keyed by the connecting
// username so a single server can serve multiple conformance-test paths
// concurrently.
package sshharness

import (
	"bufio"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/util/contract"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

// Harness is an in-process SSH server.
type Harness struct {
	Host       string
	Port       int
	PrivateKey string // PEM-encoded RSA private key clients authenticate with.

	mu       sync.Mutex
	received map[string][]string
	listener net.Listener
}

// Start launches an SSH server bound to 127.0.0.1:0 and registers cleanup with t.
func Start(t *testing.T) *Harness {
	t.Helper()

	hostKey, err := generateHostSigner()
	require.NoError(t, err, "generate host key")

	clientKeyPEM, err := generateClientKeyPEM()
	require.NoError(t, err, "generate client key")

	config := &ssh.ServerConfig{
		PublicKeyCallback: func(ssh.ConnMetadata, ssh.PublicKey) (*ssh.Permissions, error) {
			return &ssh.Permissions{}, nil
		},
		PasswordCallback: func(ssh.ConnMetadata, []byte) (*ssh.Permissions, error) {
			return &ssh.Permissions{}, nil
		},
	}
	config.AddHostKey(hostKey)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "listen")

	addr := listener.Addr().(*net.TCPAddr)
	h := &Harness{
		Host:       addr.IP.String(),
		Port:       addr.Port,
		PrivateKey: clientKeyPEM,
		received:   make(map[string][]string),
		listener:   listener,
	}
	t.Cleanup(func() { contract.IgnoreClose(listener) })

	go h.acceptLoop(t, config)
	return h
}

// Received returns a copy of the command strings received for the given username.
// Both exec payloads and SCP-uploaded file contents appear in the slice.
func (h *Harness) Received(user string) []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]string, len(h.received[user]))
	copy(out, h.received[user])
	return out
}

func (h *Harness) record(user, entry string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.received[user] = append(h.received[user], entry)
}

func (h *Harness) acceptLoop(t *testing.T, config *ssh.ServerConfig) {
	for {
		nConn, err := h.listener.Accept()
		if err != nil {
			return // listener closed
		}
		go h.handleConn(t, nConn, config)
	}
}

func (h *Harness) handleConn(t *testing.T, nConn net.Conn, config *ssh.ServerConfig) {
	defer nConn.Close()

	sshConn, chans, reqs, err := ssh.NewServerConn(nConn, config)
	if err != nil {
		t.Logf("sshharness: handshake: %v", err)
		return
	}
	defer sshConn.Close()
	user := sshConn.User()

	go ssh.DiscardRequests(reqs)

	for newCh := range chans {
		if newCh.ChannelType() != "session" {
			_ = newCh.Reject(ssh.UnknownChannelType, "only session channels supported")
			continue
		}
		ch, chReqs, err := newCh.Accept()
		if err != nil {
			t.Logf("sshharness: accept channel: %v", err)
			continue
		}
		go h.handleSession(user, ch, chReqs)
	}
}

func (h *Harness) handleSession(user string, ch ssh.Channel, reqs <-chan *ssh.Request) {
	defer ch.Close()
	for req := range reqs {
		switch req.Type {
		case "pty-req", "env", "window-change":
			if req.WantReply {
				_ = req.Reply(true, nil)
			}
		case "shell":
			if req.WantReply {
				_ = req.Reply(true, nil)
			}
			_, _ = ch.SendRequest("exit-status", false, encodeUint32(0))
			return
		case "exec":
			cmd := parseExecPayload(req.Payload)
			if req.WantReply {
				_ = req.Reply(true, nil)
			}
			if isSCPSink(cmd) {
				h.handleSCPSink(user, ch)
			} else {
				h.record(user, cmd)
			}
			_, _ = ch.SendRequest("exit-status", false, encodeUint32(0))
			return
		default:
			if req.WantReply {
				_ = req.Reply(false, nil)
			}
		}
	}
}

// isSCPSink reports whether cmd is an SCP sink invocation (i.e. receives a file).
// Terraform's remote-exec uploads its generated script via `scp -vt <path>` (or
// similar combinations of short flags). OpenSSH-style SCP may single-quote the
// program name.
func isSCPSink(cmd string) bool {
	c := strings.TrimSpace(cmd)
	if !strings.HasPrefix(c, "scp ") && !strings.HasPrefix(c, "'scp' ") {
		return false
	}
	for tok := range strings.FieldsSeq(c) {
		if strings.HasPrefix(tok, "-") && !strings.HasPrefix(tok, "--") && strings.ContainsRune(tok, 't') {
			return true
		}
	}
	return false
}

// handleSCPSink implements the minimal SCP sink-mode protocol needed to
// receive the file Terraform uploads for remote-exec. Records each file's
// contents verbatim as a recorded entry for the user.
func handleSCPAck(w io.Writer) error {
	_, err := w.Write([]byte{0})
	return err
}

func (h *Harness) handleSCPSink(user string, ch ssh.Channel) {
	br := bufio.NewReader(ch)
	if err := handleSCPAck(ch); err != nil {
		return
	}
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return
		}
		if line == "" {
			return
		}
		trimmed := strings.TrimSuffix(line, "\n")
		if trimmed == "" {
			continue
		}
		switch trimmed[0] {
		case 'C':
			// C<mode> <size> <filename>
			parts := strings.SplitN(trimmed[1:], " ", 3)
			if len(parts) != 3 {
				return
			}
			size, err := strconv.Atoi(parts[1])
			if err != nil {
				return
			}
			if err := handleSCPAck(ch); err != nil {
				return
			}
			body := make([]byte, size)
			if _, err := io.ReadFull(br, body); err != nil {
				return
			}
			// The sender terminates the file with a single NUL byte.
			if _, err := br.ReadByte(); err != nil {
				return
			}
			h.record(user, string(body))
			if err := handleSCPAck(ch); err != nil {
				return
			}
		case 'T', 'D', 'E':
			if err := handleSCPAck(ch); err != nil {
				return
			}
			if trimmed[0] == 'E' {
				return
			}
		default:
			return
		}
	}
}

// parseExecPayload extracts the command string from an SSH "exec" request
// payload (RFC 4254 §6.5): uint32 length + command bytes.
func parseExecPayload(payload []byte) string {
	if len(payload) < 4 {
		return ""
	}
	n := int(payload[0])<<24 | int(payload[1])<<16 | int(payload[2])<<8 | int(payload[3])
	if n+4 > len(payload) {
		return ""
	}
	return string(payload[4 : 4+n])
}

func encodeUint32(v uint32) []byte {
	out := make([]byte, 4)
	binary.BigEndian.PutUint32(out, v)
	return out
}

func generateHostSigner() (ssh.Signer, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("ed25519.GenerateKey: %w", err)
	}
	return ssh.NewSignerFromKey(priv)
}

// generateClientKeyPEM returns a PEM-encoded RSA private key in the classic
// "BEGIN RSA PRIVATE KEY" (PKCS#1) format that both OpenSSH and the
// pulumi-command provider accept without extra configuration.
func generateClientKeyPEM() (string, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", fmt.Errorf("rsa.GenerateKey: %w", err)
	}
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	return string(pem.EncodeToMemory(block)), nil
}
