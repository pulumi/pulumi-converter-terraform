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
	"github.com/stretchr/testify/require"
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

resource "test_resource" "indexed" {
  value = "indexed"
  list_attr = ["a", "b", "c"]
  lifecycle {
    ignore_changes = [list_attr[0]]
  }
}

resource "test_resource" "computed_list_indexed" {
  value = "computed-list"
  lifecycle {
    ignore_changes = [computed_list[0]]
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
			require.NotNil(t, example, "resource 'example' not found in state")
			assert.Equal(t, []string{"value"}, example.IgnoreChanges)

			indexed := findResource(resources, "indexed")
			require.NotNil(t, indexed, "resource 'indexed' not found in state")
			assert.Equal(t, []string{"listAttrs[0]"}, indexed.IgnoreChanges)

			computedListIndexed := findResource(resources, "computed_list_indexed")
			require.NotNil(t, computedListIndexed, "resource 'computed_list_indexed' not found in state")
			assert.Empty(t, computedListIndexed.IgnoreChanges)

			computedOnly := findResource(resources, "computed_only")
			require.NotNil(t, computedOnly, "resource 'computed_only' not found in state")
			assert.Empty(t, computedOnly.IgnoreChanges)

			bridgeComputed := findResource(resources, "bridge_computed")
			require.NotNil(t, bridgeComputed, "resource 'bridge_computed' not found in state")
			assert.Empty(t, bridgeComputed.IgnoreChanges)
		},
	})
}

// TestL2LifecycleIgnoreAll tests that ignore_changes = all is converted correctly.
//
// ignore_changes = all is valid TF syntax that ignores all attribute changes.
// https://developer.hashicorp.com/terraform/language/meta-arguments/lifecycle#ignore_changes
func TestL2LifecycleIgnoreAll(t *testing.T) {
	t.Skip("TODO[https://github.com/pulumi/pulumi-converter-terraform/issues/412]")
	t.Parallel()
	conformance.AssertConversion(t, conformance.TestCase{
		Providers: []conformance.Provider{
			{Name: "test", Factory: providers.TestProvider},
		},
		Input: map[string]string{"main.tf": `
resource "test_resource" "ignore_all" {
  value = "ignore-all"
  lifecycle {
    ignore_changes = all
  }
}
`},
		AssertState: func(t *testing.T, resources []apitype.ResourceV3) {
			t.Helper()
			ignoreAll := findResource(resources, "ignore_all")
			require.NotNil(t, ignoreAll, "resource 'ignore_all' not found in state")
			assert.Equal(t, []string{"value"}, ignoreAll.IgnoreChanges)
		},
	})
}
