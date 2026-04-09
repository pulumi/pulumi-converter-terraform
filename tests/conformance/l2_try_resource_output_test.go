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

func TestL2TryResourceOutput(t *testing.T) {
	t.Parallel()
	conformance.AssertConversion(t, conformance.TestCase{
		Providers: []conformance.Provider{
			{Name: "simple", Factory: providers.SimpleProvider},
		},
		Config: map[string]string{
			"name": "hello",
		},
		Input: map[string]string{"main.tf": `
variable "name" {
  type = string
}

resource "simple_resource" "this" {
  input_one = try(var.name, "default")
  input_two = true
}

output "missing" {
  value = try(simple_resource.this.result, 42)
}

output "present" {
  value = try(simple_resource.this.input_two, false)
}
`},
	})
}
