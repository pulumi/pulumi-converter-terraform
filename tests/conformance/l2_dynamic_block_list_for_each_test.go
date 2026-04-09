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
	"testing"

	"github.com/pulumi/pulumi-converter-terraform/pkg/testing/conformance"
	"github.com/pulumi/pulumi-converter-terraform/tests/conformance/providers"
)

// TestL2DynamicBlockListForEach reproduces a converter bug where dynamic blocks
// with list-typed for_each expressions get wrapped in entries(), which fails at
// runtime because entries() only works on maps. References inside the block also
// become doubly nested (e.g. rule.value.value.port instead of rule.value.port).
func TestL2DynamicBlockListForEach(t *testing.T) {
	// https://github.com/pulumi/pulumi-converter-terraform/issues/414
	t.Parallel()
	conformance.AssertConversion(t, conformance.TestCase{
		Providers: []conformance.Provider{
			{Name: "test", Factory: providers.TestProvider},
		},
		Input: map[string]string{"main.tf": `
variable "rules" {
  type = list(object({
    port     = number
    protocol = string
  }))
  default = [
    { port = 80, protocol = "tcp" },
    { port = 443, protocol = "tcp" }
  ]
}

resource "test_nested_resource" "example" {
  value = "test"

  dynamic "rule" {
    for_each = var.rules
    content {
      port     = rule.value.port
      protocol = rule.value.protocol
    }
  }
}

output "computed" {
  value = test_nested_resource.example.computed_value
}
`},
	})
}
