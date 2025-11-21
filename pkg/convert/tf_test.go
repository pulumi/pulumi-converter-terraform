// Copyright 2016-2023, Pulumi Corporation.
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

package convert

import (
	"testing"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
)

func TestProjectListToSingleton(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    hclwrite.Tokens
		expected hclwrite.Tokens
	}{
		{
			name: "variable",
			input: hclwrite.Tokens{
				&hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("var")},
			},
			expected: hclwrite.Tokens{
				&hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("var")},
				&hclwrite.Token{Type: hclsyntax.TokenOBrack, Bytes: []byte("[")},
				&hclwrite.Token{Type: hclsyntax.TokenNumberLit, Bytes: []byte("0")},
				&hclwrite.Token{Type: hclsyntax.TokenCBrack, Bytes: []byte("]")},
			},
		},
		{
			name: "list",
			input: hclwrite.Tokens{
				&hclwrite.Token{Type: hclsyntax.TokenOBrack, Bytes: []byte("[")},
				&hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("var")},
				&hclwrite.Token{Type: hclsyntax.TokenCBrack, Bytes: []byte("]")},
			},
			expected: hclwrite.Tokens{
				&hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("var")},
			},
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := projectListToSingleton(tt.input)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestResolveLatestProviderVersion(t *testing.T) {
	t.Parallel()
	name := impliedProvider("tfe_organization")
	provider, err := resolveRequiredProvider(name)
	assert.NoError(t, err)
	assert.Equal(t, "tfe", provider.Name)
	assert.Equal(t, "registry.terraform.io/hashicorp/tfe", provider.Source)
	// latest version resolved which unfortunately can change over time
	// expected that this test may fail in the future alongside the implicit_required_provider
	// test program
	assert.Equal(t, "~> 0.71.0", provider.Requirement.Required.String())
}
