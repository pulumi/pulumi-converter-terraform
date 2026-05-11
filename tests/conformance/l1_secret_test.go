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
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource/sig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestL1Secret asserts that the TF `sensitive` function is converted to PCL's
// `secret` intrinsic, and that the resulting stack output is encoded as a
// secret in Pulumi state.
func TestL1Secret(t *testing.T) {
	t.Parallel()
	conformance.AssertConversion(t, conformance.TestCase{
		Input: map[string]string{"main.tf": `
output "wrapped" {
  value     = sensitive("hello")
  sensitive = true
}
`},
		AssertState: func(t *testing.T, resources []apitype.ResourceV3) {
			t.Helper()
			var stack *apitype.ResourceV3
			for i := range resources {
				if resources[i].Type == "pulumi:pulumi:Stack" {
					stack = &resources[i]
					break
				}
			}
			require.NotNil(t, stack, "stack resource not found in state")

			assert.Equal(t, map[string]any{
				sig.Key:     sig.Secret,
				"plaintext": `"hello"`,
			}, stack.Outputs["wrapped"])
		},
	})
}
