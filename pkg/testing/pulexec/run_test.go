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

package pulexec

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunBasicResource(t *testing.T) {
	t.Parallel()
	tfp := &schema.Provider{
		ResourcesMap: map[string]*schema.Resource{
			"test_resource": {
				Schema: map[string]*schema.Schema{
					"value": {
						Type:     schema.TypeString,
						Required: true,
					},
					"computed_value": {
						Type:     schema.TypeString,
						Computed: true,
					},
				},
				CreateContext: func(_ context.Context, d *schema.ResourceData, _ any) diag.Diagnostics {
					d.SetId("newid")
					v := d.Get("value").(string)
					err := d.Set("computed_value", "computed_"+v)
					if err != nil {
						return diag.FromErr(err)
					}
					return nil
				},
				ReadContext: func(_ context.Context, _ *schema.ResourceData, _ any) diag.Diagnostics {
					return nil
				},
				UpdateContext: func(_ context.Context, _ *schema.ResourceData, _ any) diag.Diagnostics {
					return nil
				},
				DeleteContext: func(_ context.Context, _ *schema.ResourceData, _ any) diag.Diagnostics {
					return nil
				},
			},
		},
	}

	provider := BridgedProvider(t, "test", tfp)

	program := `resource "testRes" "test:index/resource:Resource" {
    value = "hello"
}
output "value" {
    value = testRes.value
}
output "computedValue" {
    value = testRes.computedValue
}
`

	outputs := Run(t, []Provider{{Name: "test", Info: provider}}, map[string]string{"main.pp": program}, nil, nil)

	require.Equal(t, map[string]string{
		"value":         "hello",
		"computedValue": "computed_hello",
	}, outputs.Outputs)
}

func TestBridgedProviderValidates(t *testing.T) {
	t.Parallel()
	tfp := &schema.Provider{
		ResourcesMap: map[string]*schema.Resource{
			"test_thing": {
				Schema: map[string]*schema.Schema{
					"name": {
						Type:     schema.TypeString,
						Required: true,
					},
				},
				CreateContext: func(_ context.Context, d *schema.ResourceData, _ any) diag.Diagnostics {
					d.SetId("newid")
					return nil
				},
				ReadContext: func(_ context.Context, _ *schema.ResourceData, _ any) diag.Diagnostics {
					return nil
				},
				UpdateContext: func(_ context.Context, _ *schema.ResourceData, _ any) diag.Diagnostics {
					return nil
				},
				DeleteContext: func(_ context.Context, _ *schema.ResourceData, _ any) diag.Diagnostics {
					return nil
				},
			},
		},
	}

	provider := BridgedProvider(t, "test", tfp)

	assert.Equal(t, "test", provider.Name)
	assert.Equal(t, "0.0.1", provider.Version)
}
