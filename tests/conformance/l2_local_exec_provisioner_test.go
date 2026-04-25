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
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/pulumi/pulumi-converter-terraform/pkg/testing/conformance"
	"github.com/pulumi/pulumi-converter-terraform/tests/conformance/providers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var installPulumiCommandPlugin = func(t *testing.T) {
	var err error
	var once sync.Once
	once.Do(func() {
		cmd := exec.CommandContext(t.Context(), "pulumi", "plugin", "install", "resource", "command")
		err = cmd.Run()
	})
	require.NoError(t, err)
}

// TestL2LocalExecProvisioner verifies that a local-exec provisioner referencing
// "self.X" on its parent resource is correctly converted and runs end to end.
//
// Each path writes to a filename derived from var.conformance_kind (set to "tf"
// and "pulumi" by the harness) so the TF and Pulumi runs do not race on a
// shared output file.
func TestL2LocalExecProvisioner(t *testing.T) {
	t.Parallel()

	installPulumiCommandPlugin(t)

	outDir := t.TempDir()

	conformance.AssertConversion(t, conformance.TestCase{
		Providers: []conformance.Provider{
			{Name: "test", Factory: providers.TestProvider},
		},
		Config: map[string]string{"output_path": outDir},
		Input: map[string]string{"main.tf": `
variable "conformance_kind" {
  type = string
}

variable "output_path" {
  type = string
}

resource "test_resource" "example" {
  value = "hello"

  provisioner "local-exec" {
    command = "printf %s \"${self.computed_value}\" > \"${var.output_path}/${var.conformance_kind}.txt\""
  }
}

output "value" {
  value = test_resource.example.value
}
`},
	})

	for _, kind := range []string{"tf", "pulumi"} {
		got, err := os.ReadFile(filepath.Join(outDir, kind+".txt"))
		require.NoError(t, err, "reading %s output", kind)
		assert.Equal(t, "computed_hello", string(got), "%s provisioner output", kind)
	}
}
