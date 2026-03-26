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
	"github.com/stretchr/testify/assert"
)

func TestL2Lifecycle(t *testing.T) {
	t.Parallel()
	conformance.AssertConversion(t, conformance.TestCase{
		Providers: []conformance.Provider{
			{Name: "test", Factory: providers.TestProvider},
		},
		Input: map[string]string{"main.tf": `
resource "test_resource" "example" {
  value = "hello"
  lifecycle {
    ignore_changes = [value]
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
		},
	})
}
