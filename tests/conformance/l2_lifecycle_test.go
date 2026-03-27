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
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfbridge"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/stretchr/testify/assert"
)

func TestL2Lifecycle(t *testing.T) {
	t.Parallel()
	conformance.AssertConversion(t, conformance.TestCase{
		Providers: []conformance.Provider{
			{
				Name:    "test",
				Factory: providers.TestProvider,
				EditInfo: func(info *tfbridge.ProviderInfo) {
					r := info.Resources["test_tagged_resource"]
					r.Fields = map[string]*tfbridge.SchemaInfo{
						"marked_as_computed_only": {
							MarkAsComputedOnly: ref(true),
						},
					}
				},
			},
		},
		Input: map[string]string{"main.tf": `
resource "test_resource" "example" {
  value = "hello"
  lifecycle {
    ignore_changes = [value]
  }
}

resource "test_resource" "computed_only" {
  value = "world"
  lifecycle {
    ignore_changes = [computed_value]
  }
}

resource "test_tagged_resource" "bridge_computed" {
  value = "tagged"
  lifecycle {
    ignore_changes = [marked_as_computed_only]
  }
}

output "result" {
  value = test_resource.example.computed_value
}
`},
		AssertState: func(t *testing.T, resources []apitype.ResourceV3) {
			t.Helper()
			example := findResource(resources, "example")
			if example == nil {
				t.Fatal("resource 'example' not found in state")
			}
			assert.Equal(t, []string{"value"}, example.IgnoreChanges)

			computedOnly := findResource(resources, "computed_only")
			if computedOnly == nil {
				t.Fatal("resource 'computed_only' not found in state")
			}
			assert.Empty(t, computedOnly.IgnoreChanges)

			bridgeComputed := findResource(resources, "bridge_computed")
			if bridgeComputed == nil {
				t.Fatal("resource 'bridge_computed' not found in state")
			}
			assert.Empty(t, bridgeComputed.IgnoreChanges)
		},
	})
}
