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

package conformance

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/pulumi/pulumi-converter-terraform/pkg/testing/conformance"
	"github.com/pulumi/pulumi-converter-terraform/pkg/testing/sshharness"
	"github.com/pulumi/pulumi-converter-terraform/tests/conformance/providers"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource/sig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestL2RemoteExecProvisioner exercises three remote-exec attribute forms —
// `inline`, `script`, and `scripts` — each against its own SSH server. Both
// the TF and Pulumi paths dial the per-resource server but authenticate with
// different usernames (driven by conformance_kind, auto-injected by the
// harness) so each path's traffic is recorded separately.
func TestL2RemoteExecProvisioner(t *testing.T) {
	t.Parallel()

	srvInline := sshharness.Start(t)
	srvScript := sshharness.Start(t)
	srvScripts := sshharness.Start(t)

	mainTF := `
variable "conformance_kind" {
  type = string
}

variable "ssh_pk" {
  type = string
}

variable "ssh_port_inline" {
  type = number
}

variable "ssh_port_script" {
  type = number
}

variable "ssh_port_scripts" {
  type = number
}

resource "test_resource" "example_inline" {
  value = "hello"
  provisioner "remote-exec" {
    connection {
      host        = "127.0.0.1"
      port        = var.ssh_port_inline
      user        = var.conformance_kind
      private_key = var.ssh_pk
    }
    inline = ["echo ${self.computed_value}"]
  }
}

resource "test_resource" "example_script" {
  value = "world"
  provisioner "remote-exec" {
    connection {
      host        = "127.0.0.1"
      port        = var.ssh_port_script
      user        = var.conformance_kind
      private_key = var.ssh_pk
    }
    script = "./hello.sh"
  }
}

resource "test_resource" "example_scripts" {
  value = "many"
  provisioner "remote-exec" {
    connection {
      host        = "127.0.0.1"
      port        = var.ssh_port_scripts
      user        = var.conformance_kind
      private_key = var.ssh_pk
    }
    scripts = ["./a.sh", "./b.sh"]
  }
}
`

	conformance.AssertConversion(t, conformance.TestCase{
		Providers: []conformance.Provider{
			{Name: "test", Factory: providers.TestProvider},
		},
		Config: map[string]string{
			"ssh_pk":           srvInline.PrivateKey,
			"ssh_port_inline":  strconv.Itoa(srvInline.Port),
			"ssh_port_script":  strconv.Itoa(srvScript.Port),
			"ssh_port_scripts": strconv.Itoa(srvScripts.Port),
		},
		AssertState: func(t *testing.T, resources []apitype.ResourceV3) {
			t.Helper()
			assertInlineCommandInputs(t, resources, srvInline.Port, srvInline.PrivateKey)
		},
		Input: map[string]string{
			"main.tf":  mainTF,
			"hello.sh": "#!/bin/sh\necho hello\n",
			"a.sh":     "#!/bin/sh\necho a\n",
			"b.sh":     "#!/bin/sh\necho b\n",
		},
	})

	// Verify each per-resource SSH server saw the expected payloads. TF's
	// remote-exec uploads scripts via SCP; Pulumi's CopyToRemote uses SFTP;
	// inline runs straight via exec. In all cases the marker text appears
	// somewhere in the recorded payloads for both sessions.
	checks := []struct {
		name   string
		srv    *sshharness.Harness
		marker string
	}{
		{"inline", srvInline, "echo computed_hello"},
		{"script", srvScript, "echo hello"},
		{"scripts", srvScripts, "echo a"},
	}
	for _, c := range checks {
		for _, kind := range []string{"tf", "pulumi"} {
			recorded := c.srv.Received(kind)
			joined := strings.Join(recorded, "\n")
			assert.Contains(t, joined, c.marker,
				"%s server: %q session expected marker %q in %#v",
				c.name, kind, c.marker, recorded)
		}
	}

	// `scripts` uploads two files; check the second one too.
	for _, kind := range []string{"tf", "pulumi"} {
		recorded := srvScripts.Received(kind)
		joined := strings.Join(recorded, "\n")
		assert.Contains(t, joined, "echo b",
			"scripts server: %q session expected second-script marker in %#v", kind, recorded)
	}
}

// assertInlineCommandInputs pins the inputs of the inline command resource so
// any drift in connection-secret handling or `create` formatting is flagged.
func assertInlineCommandInputs(
	t *testing.T, resources []apitype.ResourceV3, port int, privateKey string,
) {
	t.Helper()
	var provisioner apitype.ResourceV3
	for _, r := range resources {
		if r.URN.Name() == "exampleInlineProvisioner0" {
			provisioner = r
			break
		}
	}

	innerPK, err := json.Marshal(privateKey)
	require.NoError(t, err)
	connection := map[string]any{
		"dialErrorLimit": 10,
		"host":           "127.0.0.1",
		"perDialTimeout": 15,
		"port":           port,
		"privateKey": map[string]any{
			sig.Key:     sig.Secret,
			"plaintext": string(innerPK),
		},
		"user": "pulumi",
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	require.NoError(t, enc.Encode(connection))

	assert.Equal(t, map[string]any{
		"connection": map[string]any{
			sig.Key:     sig.Secret,
			"plaintext": strings.TrimRight(buf.String(), "\n"),
		},
		"create": "echo computed_hello",
	}, provisioner.Inputs)
}
