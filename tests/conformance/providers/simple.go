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
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// SimpleProvider returns a provider matching the "simple" schema used by
// pkg/convert/testdata/mappings/simple.json, with TypeBool for input_two
// so that `input_two = var.bool_in` works at runtime.
func SimpleProvider() *schema.Provider {
	return &schema.Provider{
		ResourcesMap: map[string]*schema.Resource{
			"simple_resource": {
				Schema: map[string]*schema.Schema{
					"input_one": {
						Type:     schema.TypeString,
						Optional: true,
					},
					"input_two": {
						Type:     schema.TypeBool,
						Optional: true,
					},
					"result": {
						Type:     schema.TypeString,
						Computed: true,
					},
				},
				CreateContext: func(_ context.Context, d *schema.ResourceData, _ any) diag.Diagnostics {
					d.SetId("simple-id")
					one, _ := d.Get("input_one").(string)
					two := d.Get("input_two").(bool)
					if err := d.Set("result", fmt.Sprintf("%s-%t", one, two)); err != nil {
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
			"simple_another_resource": {
				Schema: map[string]*schema.Schema{
					"input_one": {
						Type:     schema.TypeString,
						Optional: true,
					},
					"result": {
						Type:     schema.TypeString,
						Computed: true,
					},
				},
				CreateContext: func(_ context.Context, d *schema.ResourceData, _ any) diag.Diagnostics {
					d.SetId("another-id")
					one, _ := d.Get("input_one").(string)
					if err := d.Set("result", "another-"+one); err != nil {
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
		DataSourcesMap: map[string]*schema.Resource{
			"simple_data_source": {
				Schema: map[string]*schema.Schema{
					"input_one": {
						Type:     schema.TypeString,
						Optional: true,
					},
					"input_two": {
						Type:     schema.TypeBool,
						Optional: true,
					},
					"result": {
						Type:     schema.TypeString,
						Computed: true,
					},
				},
				ReadContext: func(_ context.Context, d *schema.ResourceData, _ any) diag.Diagnostics {
					d.SetId("data-id")
					one, _ := d.Get("input_one").(string)
					if err := d.Set("result", "data-"+one); err != nil {
						return diag.FromErr(err)
					}
					return nil
				},
			},
		},
	}
}
