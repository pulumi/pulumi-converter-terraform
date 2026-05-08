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

// dynamicTimeoutsHCL exercises the `dynamic "timeouts"` pattern used by
// terraform-aws-modules/eks (fargate-profile, managed-node-group) where
// timeouts are wired through a per-module variable and a one-element
// for_each so the block is included only when values are supplied.
const dynamicTimeoutsHCL = `
variable "timeout_create" {
  type    = string
  default = "5m"
}

variable "timeout_delete" {
  type    = string
  default = "30s"
}

resource "test_resource" "with_dynamic_timeouts" {
  value = "x"
  dynamic "timeouts" {
    for_each = [{ create = var.timeout_create, delete = var.timeout_delete }]
    content {
      create = lookup(timeouts.value, "create", null)
      delete = lookup(timeouts.value, "delete", null)
    }
  }
}
`

// TestL2TimeoutsDynamic asserts that a `dynamic "timeouts"` block is converted
// to a Pulumi `customTimeouts` resource option.
//
// Today the converter emits the dynamic block as a regular resource attribute,
// producing PCL that fails to bind with `unsupported attribute 'timeouts'`.
//
// This case runs without overriding the timeout vars — the resource picks up
// the defaults declared in the HCL.
func TestL2TimeoutsDynamic(t *testing.T) {
	t.Parallel()
	conformance.AssertConversion(t, conformance.TestCase{
		Providers: []conformance.Provider{
			{Name: "test", Factory: providers.TestProvider},
		},
		Input: map[string]string{"main.tf": dynamicTimeoutsHCL},
		AssertState: func(t *testing.T, resources []apitype.ResourceV3) {
			t.Helper()
			r := findResource(resources, "with_dynamic_timeouts")
			require.NotNil(t, r, "resource 'with_dynamic_timeouts' not found in state")
			assert.Equal(t, &resource.CustomTimeouts{
				Create: 5 * 60.0,
				Delete: 30.0,
			}, r.CustomTimeouts)
		},
	})
}

// TestL2TimeoutsDynamicWithConfig is the same `dynamic "timeouts"` program but
// with the timeout vars overridden via Config. Asserts the substituted
// for_each tuple element is evaluated at deploy time, not baked in at
// conversion time — a different config produces a different CustomTimeouts.
func TestL2TimeoutsDynamicWithConfig(t *testing.T) {
	t.Parallel()
	conformance.AssertConversion(t, conformance.TestCase{
		Providers: []conformance.Provider{
			{Name: "test", Factory: providers.TestProvider},
		},
		Config: map[string]string{
			"timeout_create": "10m",
			"timeout_delete": "1m",
		},
		Input: map[string]string{"main.tf": dynamicTimeoutsHCL},
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
}
