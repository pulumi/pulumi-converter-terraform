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

package tfexec

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyBasic(t *testing.T) {
	t.Parallel()
	provider := &schema.Provider{
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
		},
	}

	driver := NewDriver(t, []Provider{{Name: "test", Provider: provider}})

	outputs := driver.Apply(t, map[string]string{
		"main.tf": `
resource "test_resource" "example" {
  value = "hello"
}

output "value" {
  value = test_resource.example.value
}

output "computed_value" {
  value = test_resource.example.computed_value
}
`,
	}, nil)

	require.Len(t, outputs, 2)
	assert.Equal(t, map[string]string{
		"value":          "hello",
		"computed_value": "computed_hello",
	}, outputs)
}
