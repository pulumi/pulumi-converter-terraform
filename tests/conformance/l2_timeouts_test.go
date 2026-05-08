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
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestL2Timeouts asserts that a TF resource `timeouts` block is converted
// to a Pulumi `customTimeouts` resource option.
//
// Today the converter silently drops `timeouts` blocks, so the generated PCL
// has no `customTimeouts` option and the resource's CustomTimeouts is nil.
func TestL2Timeouts(t *testing.T) {
	t.Parallel()
	conformance.AssertConversion(t, conformance.TestCase{
		Providers: []conformance.Provider{
			{Name: "test", Factory: providers.TestProvider},
		},
		Input: map[string]string{"main.tf": `
resource "test_resource" "with_timeouts" {
  value = "x"
  timeouts {
    create = "5m"
    update = "10m"
    delete = "30s"
  }
}
`},
		AssertState: func(t *testing.T, resources []apitype.ResourceV3) {
			t.Helper()
			r := findResource(resources, "with_timeouts")
			require.NotNil(t, r, "resource 'with_timeouts' not found in state")
			assert.Equal(t, &resource.CustomTimeouts{
				Create: 5 * 60.0,
				Update: 10 * 60.0,
				Delete: 30.0,
			}, r.CustomTimeouts)
		},
	})
}

// TestL2TimeoutsDynamic asserts that a `dynamic "timeouts"` block is converted
// to a Pulumi `customTimeouts` resource option.
//
// The HCL mirrors the pattern used by terraform-aws-modules/eks (fargate-profile,
// managed-node-group): an object-typed variable with a for_each that emits zero
// or one elements depending on whether the variable is null.
//
// Subtests cover both branches:
//   - "unset": var.timeouts defaults to null — the dynamic block does not fire
//     and the resource has no CustomTimeouts.
//   - "set":   var.timeouts defaults to an object — the dynamic block fires once
//     and CustomTimeouts reflects the object.
func TestL2TimeoutsDynamic(t *testing.T) {
	t.Parallel()

	const program = `
variable "timeouts" {
  type = object({
    create = string
    delete = string
  })
  default = null
}

resource "test_resource" "with_dynamic_timeouts" {
  value = "x"
  dynamic "timeouts" {
    for_each = var.timeouts != null ? [var.timeouts] : []
    content {
      create = lookup(timeouts.value, "create", null)
      delete = lookup(timeouts.value, "delete", null)
    }
  }
}
`

	t.Run("unset", func(t *testing.T) {
		t.Parallel()
		conformance.AssertConversion(t, conformance.TestCase{
			Providers: []conformance.Provider{
				{Name: "test", Factory: providers.TestProvider},
			},
			Input: map[string]string{"main.tf": program},
			AssertState: func(t *testing.T, resources []apitype.ResourceV3) {
				t.Helper()
				r := findResource(resources, "with_dynamic_timeouts")
				require.NotNil(t, r, "resource 'with_dynamic_timeouts' not found in state")
				assert.Nil(t, r.CustomTimeouts,
					"null var.timeouts should produce no customTimeouts")
			},
		})
	})

	t.Run("set", func(t *testing.T) {
		t.Parallel()
		conformance.AssertConversion(t, conformance.TestCase{
			Config: map[string]string{"timeouts": `{"create":"10m","delete":"1m"}`},
			Providers: []conformance.Provider{
				{Name: "test", Factory: providers.TestProvider},
			},
			Input: map[string]string{
				"main.tf": program,
			},
			AssertState: func(t *testing.T, resources []apitype.ResourceV3) {
				t.Helper()
				r := findResource(resources, "with_dynamic_timeouts")
				require.NotNil(t, r, "resource 'with_dynamic_timeouts' not found in state")
				assert.Equal(t, &resource.CustomTimeouts{
					Create: 10 * 60.0,
					Delete: 60.0,
				}, r.CustomTimeouts)
			},
		})
	})
}
