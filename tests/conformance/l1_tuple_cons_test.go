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
)

func TestL1TupleCons(t *testing.T) {
	t.Parallel()
	conformance.AssertConversion(t, conformance.TestCase{
		Config: map[string]string{
			"item": "dynamic",
		},
		Input: map[string]string{"main.tf": `
variable "item" {
  type = string
}

output "static_list" {
  value = ["a", "b", "c"]
}

output "list_with_var" {
  value = ["first", var.item, "last"]
}

output "nested_list" {
  value = [["x", "y"], ["z"]]
}
`},
	})
}
