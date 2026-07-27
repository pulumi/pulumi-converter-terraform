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

// TestL2ForEachStringKey exercises the canonical TF idiom for using a set of
// strings as a resource's `for_each`: wrapping a list with `toset(...)`.
//
// In TF, `for_each = toset(<list>)` produces a string-keyed map of instances
// where each.key == each.value, so `test_resource.mapped["alpha"]` is a valid
// lookup. Naively converting this to PCL emits
// `range = invoke("std:index:toset", ...).result`, which types as a list/set
// and forces the resource to be indexed by integer — `mapped["alpha"]` then
// fails to bind ("a number is required"). The converter sidesteps this by
// rewriting `toset(<x>)` into `{ for entry in <x> : entry => entry }` so PCL
// sees a string-keyed map directly.
func TestL2ForEachStringKey(t *testing.T) {
	// https://github.com/pulumi/pulumi-converter-terraform/issues/228
	t.Parallel()
	conformance.AssertConversion(t, conformance.TestCase{
		Providers: []conformance.Provider{
			{Name: "test", Factory: providers.TestProvider},
		},
		Input: map[string]string{"main.tf": `
resource "test_resource" "mapped" {
  for_each = toset(["alpha", "beta"])
  value    = each.value
}

output "alpha" {
  value = test_resource.mapped["alpha"].computed_value
}

output "beta" {
  value = test_resource.mapped["beta"].computed_value
}
`},
	})
}
