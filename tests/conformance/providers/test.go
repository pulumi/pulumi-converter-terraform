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

package providers

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// TestProvider returns a minimal provider with a single resource that has a
// required string input and a computed string output derived from the input.
func TestProvider() *schema.Provider {
	return &schema.Provider{
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
					d.SetId("test-id")
					v := d.Get("value").(string)
					if err := d.Set("computed_value", "computed_"+v); err != nil {
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
			// test_tagged_resource models the tags_all pattern: a field that is
			// Optional+Computed in the TF schema but marked as computed-only by
			// the bridge via MarkAsComputedOnly.
			"test_tagged_resource": {
				Schema: map[string]*schema.Schema{
					"value": {
						Type:     schema.TypeString,
						Required: true,
					},
					"marked_as_computed_only": {
						Type:     schema.TypeString,
						Optional: true,
						Computed: true,
					},
				},
				CreateContext: func(_ context.Context, d *schema.ResourceData, _ any) diag.Diagnostics {
					d.SetId("tagged-id")
					if _, ok := d.GetOk("marked_as_computed_only"); !ok {
						if err := d.Set("marked_as_computed_only", "default"); err != nil {
							return diag.FromErr(err)
						}
					}
					return nil
				},
			},
		},
		DataSourcesMap: map[string]*schema.Resource{
			"test_data": {
				Schema: map[string]*schema.Schema{
					"input": {
						Type:     schema.TypeString,
						Required: true,
					},
					"result": {
						Type:     schema.TypeString,
						Computed: true,
					},
				},
				ReadContext: func(_ context.Context, d *schema.ResourceData, _ any) diag.Diagnostics {
					d.SetId("data-id")
					v := d.Get("input").(string)
					if err := d.Set("result", "result_"+v); err != nil {
						return diag.FromErr(err)
					}
					return nil
				},
			},
		},
	}
}
