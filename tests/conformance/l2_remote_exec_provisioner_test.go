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

// TestL2RemoteExecProvisioner runs a remote-exec provisioner against an
// in-process SSH server. Both the TF and Pulumi paths dial the same server
// but authenticate with different usernames (driven by conformance_kind,
// auto-injected by the harness) so the server can record each path's
// commands separately.
func TestL2RemoteExecProvisioner(t *testing.T) {
	t.Parallel()

	srv := sshharness.Start(t)

	conformance.AssertConversion(t, conformance.TestCase{
		Providers: []conformance.Provider{
			{Name: "test", Factory: providers.TestProvider},
		},
		Config: map[string]string{
			"ssh_host":        srv.Host,
			"ssh_port":        strconv.Itoa(srv.Port),
			"ssh_private_key": srv.PrivateKey,
		},
		AssertState: func(t *testing.T, resources []apitype.ResourceV3) {
			t.Helper()
			var provisioner apitype.ResourceV3
			for _, r := range resources {
				if r.URN.Name() == "exampleProvisioner0" {
					provisioner = r
					break
				}
			}

			// The privateKey field is wrapped as a nested secret whose plaintext
			// is the JSON-encoded PEM string. Build the expected connection
			// plaintext via json.Marshal so number/string typing and escape
			// sequences match what pulumi-command emits.
			innerPK, err := json.Marshal(srv.PrivateKey)
			require.NoError(t, err)
			connection := map[string]any{
				"dialErrorLimit": 10,
				"host":           "127.0.0.1",
				"perDialTimeout": 15,
				"port":           srv.Port,
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
		},
		Input: map[string]string{"main.tf": `
variable "conformance_kind" {
  type = string
}

variable "ssh_host" {
  type = string
}

variable "ssh_port" {
  type = number
}

variable "ssh_private_key" {
  type = string
}

resource "test_resource" "example" {
  value = "hello"

  provisioner "remote-exec" {
    connection {
      host        = var.ssh_host
      port        = var.ssh_port
      user        = var.conformance_kind
      private_key = var.ssh_private_key
    }
    inline = ["echo ${self.computed_value}"]
  }
}

output "value" {
  value = test_resource.example.value
}
`},
	})

	// TF's remote-exec uploads its generated script via SCP and then exec's it,
	// so the recorded payload is the script body. Pulumi's command:remote:Command
	// just exec's the create string directly. Either way, the echo command must
	// appear somewhere in the recorded payloads for each session.
	for _, kind := range []string{"tf", "pulumi"} {
		recorded := srv.Received(kind)
		joined := strings.Join(recorded, "\n")
		assert.Contains(t, joined, "echo computed_hello",
			"commands recorded for %q session: %#v", kind, recorded)
	}
}
