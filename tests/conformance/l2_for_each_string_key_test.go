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

// TestL2ForEachStringKey reproduces a converter bug where for_each with a set of strings
// produces PCL that uses range-based indexing. TF allows string-key access like
// test_resource.mapped["alpha"], but the converted PCL treats the result as a list,
// causing "a number is required" at runtime.
func TestL2ForEachStringKey(t *testing.T) {
	// https://github.com/pulumi/pulumi-converter-terraform/issues/228
	t.Skip("for_each with string keys produces range-based indexing")
	t.Parallel()
	conformance.AssertConversion(t, conformance.TestCase{
		Providers: []conformance.Provider{
			{Name: "test", Factory: providers.TestProvider},
		},
		HCL: `
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
`,
	})
}
