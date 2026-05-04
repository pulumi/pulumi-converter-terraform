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

// Package sshcontainer launches an OpenSSH server inside a Docker container
// for use by remote-exec conformance tests. Tests fail (rather than skip)
// when Docker is unavailable.
package sshcontainer

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/crypto/ssh"
)

// Container exposes the running SSH server.
type Container struct {
	t          *testing.T
	c          testcontainers.Container
	host       string
	port       int
	user       string
	privateKey string
	publicKey  string
	signer     ssh.Signer
}

const (
	containerImage = "lscr.io/linuxserver/openssh-server:latest"
	internalPort   = "2222/tcp"
	sshUser        = "test"
)

// Start boots the container, generates an ed25519 keypair, configures the server
// to authorize that key for the configured user, and waits until the SSH port
// accepts a successful login.
func Start(t *testing.T) *Container {
	t.Helper()

	priv, pub := generateED25519KeyPair(t)

	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        containerImage,
		ExposedPorts: []string{internalPort},
		Env: map[string]string{
			"PUID":            "1000",
			"PGID":            "1000",
			"USER_NAME":       sshUser,
			"PUBLIC_KEY":      pub,
			"SUDO_ACCESS":     "false",
			"PASSWORD_ACCESS": "false",
		},
		WaitingFor: wait.ForListeningPort(internalPort).WithStartupTimeout(90 * time.Second),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "failed to start ssh container (is Docker running?)")

	t.Cleanup(func() {
		_ = container.Terminate(context.Background())
	})

	host, err := container.Host(ctx)
	require.NoError(t, err)
	mapped, err := container.MappedPort(ctx, internalPort)
	require.NoError(t, err)
	port, err := strconv.Atoi(mapped.Port())
	require.NoError(t, err)

	signer, err := ssh.ParsePrivateKey([]byte(priv))
	require.NoError(t, err)

	c := &Container{
		t: t, c: container,
		host: host, port: port, user: sshUser,
		privateKey: priv, publicKey: pub, signer: signer,
	}

	// The image's WaitingFor reports the port open as soon as sshd binds, but
	// authorized_keys is written by an init script that may run shortly after.
	// Poll for a successful auth handshake before returning.
	require.Eventually(t, func() bool {
		client, err := c.dial(2 * time.Second)
		if err != nil {
			return false
		}
		_ = client.Close()
		return true
	}, 60*time.Second, 1*time.Second, "ssh server never accepted our key")

	return c
}

// Host returns the address on which the SSH server is listening.
func (c *Container) Host() string { return c.host }

// Port returns the host-mapped TCP port for the SSH server.
func (c *Container) Port() int { return c.port }

// User returns the SSH login user provisioned in the container.
func (c *Container) User() string { return c.user }

// PrivateKeyPEM returns the PEM-encoded private key trusted by the container.
func (c *Container) PrivateKeyPEM() string { return c.privateKey }

// PublicKey returns the OpenSSH-formatted public key configured in the container.
func (c *Container) PublicKey() string { return c.publicKey }

// HostKey reads the container's SSH host public key (ed25519) so callers can
// pin it via the connection's hostKey property.
func (c *Container) HostKey() string {
	c.t.Helper()
	ctx := context.Background()
	rc, err := c.c.CopyFileFromContainer(ctx, "/config/ssh_host_keys/ssh_host_ed25519_key.pub")
	require.NoError(c.t, err)
	defer rc.Close()
	raw, err := io.ReadAll(rc)
	require.NoError(c.t, err)
	return strings.TrimSpace(string(raw))
}

// Exec runs a command on the container via SSH and returns its combined output.
func (c *Container) Exec(command string) (string, error) {
	client, err := c.dial(10 * time.Second)
	if err != nil {
		return "", err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	out, err := session.CombinedOutput(command)
	return string(out), err
}

// MustExec is Exec but fails the test on error.
func (c *Container) MustExec(command string) string {
	c.t.Helper()
	out, err := c.Exec(command)
	require.NoError(c.t, err, "ssh exec %q failed: %s", command, out)
	return out
}

func (c *Container) dial(timeout time.Duration) (*ssh.Client, error) {
	cfg := &ssh.ClientConfig{
		User:            c.user,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(c.signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // test container
		Timeout:         timeout,
	}
	addr := net.JoinHostPort(c.host, strconv.Itoa(c.port))
	return ssh.Dial("tcp", addr, cfg)
}

func generateED25519KeyPair(t *testing.T) (privatePEM, publicAuthorized string) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	pemBlock, err := ssh.MarshalPrivateKey(priv, "pulumi-converter-conformance")
	require.NoError(t, err)
	privatePEM = string(pem.EncodeToMemory(pemBlock))

	sshPub, err := ssh.NewPublicKey(pub)
	require.NoError(t, err)
	publicAuthorized = string(ssh.MarshalAuthorizedKey(sshPub))
	return privatePEM, publicAuthorized
}

// ConnectionVars returns a string-keyed map suitable to pass into TestCase.Config
// for the conformance harness. Each value is a literal string.
func (c *Container) ConnectionVars() map[string]string {
	return map[string]string{
		"ssh_host":        c.host,
		"ssh_port":        strconv.Itoa(c.port),
		"ssh_user":        c.user,
		"ssh_private_key": c.privateKey,
	}
}
