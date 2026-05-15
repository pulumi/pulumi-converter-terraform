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

// TestL2DynamicBlockConditionalGate exercises the TF idiom
// `for_each = <cond> ? [1] : []` on a `dynamic` block, which is the common way
// to make a single-instance nested block conditional on a flag.
//
// Naively converting this to PCL emits a for-expression over a tuple-of-number
// that PCL's binder can't bind cleanly (see #228 for the original report). The
// converter detects this exact pattern — both branches are tuple literals, the
// truthy branch has one element, the falsy branch is empty — and rewrites the
// dynamic block to a conditional that produces either a one-element list of
// the converted content or an empty list, sidestepping the binder issue.
func TestL2DynamicBlockConditionalGate(t *testing.T) {
	t.Parallel()
	conformance.AssertConversion(t, conformance.TestCase{
		Providers: []conformance.Provider{
			{Name: "test", Factory: providers.TestProvider},
		},
		Input: map[string]string{"main.tf": `
variable "include_rule" {
  type    = bool
  default = true
}

resource "test_nested_resource" "example" {
  value = "test"

  dynamic "rule" {
    for_each = var.include_rule ? [80] : []
    content {
      port     = rule.value
      protocol = "tcp"
    }
  }
}

output "computed" {
  value = test_nested_resource.example.computed_value
}
`},
	})
}
