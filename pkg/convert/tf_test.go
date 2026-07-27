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
	"context"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/pulumi/terraform/pkg/addrs"
	"github.com/pulumi/terraform/pkg/getproviders"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := projectListToSingleton(tt.input)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

type TestRegistrySource struct{}

func (s *TestRegistrySource) AvailableVersions(
	ctx context.Context, provider addrs.Provider,
) (getproviders.VersionList, getproviders.Warnings, error) {
	return getproviders.VersionList{
		getproviders.Version{
			Major: 0,
			Minor: 71,
			Patch: 0,
		},
	}, getproviders.Warnings{}, nil
}

func (s *TestRegistrySource) ForDisplay(provider addrs.Provider) string {
	return "registry.terraform.io/hashicorp/tfe"
}

func (s *TestRegistrySource) PackageMeta(
	ctx context.Context, provider addrs.Provider, version getproviders.Version, target getproviders.Platform,
) (getproviders.PackageMeta, error) {
	return getproviders.PackageMeta{
		Version: version,
	}, nil
}

func TestCustomTimeoutsTokens(t *testing.T) {
	t.Parallel()

	thirtyS := hclwrite.Tokens{
		&hclwrite.Token{Type: hclsyntax.TokenOQuote, Bytes: []byte(`"`)},
		&hclwrite.Token{Type: hclsyntax.TokenQuotedLit, Bytes: []byte("30s")},
		&hclwrite.Token{Type: hclsyntax.TokenCQuote, Bytes: []byte(`"`)},
	}
	oneMin := hclwrite.Tokens{
		&hclwrite.Token{Type: hclsyntax.TokenOQuote, Bytes: []byte(`"`)},
		&hclwrite.Token{Type: hclsyntax.TokenQuotedLit, Bytes: []byte("1m")},
		&hclwrite.Token{Type: hclsyntax.TokenCQuote, Bytes: []byte(`"`)},
	}

	cases := []struct {
		name           string
		create, update hclwrite.Tokens
		expected       string
	}{
		{name: "neither", create: nil, update: nil, expected: ""},
		{name: "create only", create: thirtyS, update: nil, expected: `{
  create = "30s"
}`},
		{name: "update only", create: nil, update: thirtyS, expected: `{
  update = "30s"
}`},
		{name: "both same value", create: thirtyS, update: thirtyS, expected: `{
  create = "30s"
  update = "30s"
}`},
		{name: "both differ", create: thirtyS, update: oneMin, expected: `{
  create = "30s"
  update = "1m"
}`},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := customTimeoutsTokens(tt.create, tt.update)
			if tt.expected == "" {
				assert.Nil(t, got)
				return
			}
			f := hclwrite.NewEmptyFile()
			f.Body().SetAttributeRaw("v", got)
			assert.Equal(t, "v = "+tt.expected+"\n", string(hclwrite.Format(f.Bytes())))
		})
	}
}

func parseExprForTest(t *testing.T, src string) hclsyntax.Expression {
	t.Helper()
	expr, diags := hclsyntax.ParseExpression([]byte(src), "test.hcl", hcl.Pos{Line: 1, Column: 1})
	require.False(t, diags.HasErrors(), "parse failed: %s", diags)
	return expr
}

func TestDetectTosetForEach(t *testing.T) {
	t.Parallel()

	t.Run("matches toset(list)", func(t *testing.T) {
		t.Parallel()
		inner, ok := detectTosetForEach(parseExprForTest(t, `toset(["a", "b"])`))
		require.True(t, ok)
		tuple, isTuple := inner.(*hclsyntax.TupleConsExpr)
		require.True(t, isTuple, "expected tuple, got %T", inner)
		assert.Len(t, tuple.Exprs, 2)
	})

	t.Run("rejects other function", func(t *testing.T) {
		t.Parallel()
		_, ok := detectTosetForEach(parseExprForTest(t, `tolist(["a"])`))
		assert.False(t, ok)
	})

	t.Run("rejects wrong arity", func(t *testing.T) {
		t.Parallel()
		_, ok := detectTosetForEach(parseExprForTest(t, `toset(["a"], ["b"])`))
		assert.False(t, ok)
	})

	t.Run("rejects non-function expression", func(t *testing.T) {
		t.Parallel()
		_, ok := detectTosetForEach(parseExprForTest(t, `var.x`))
		assert.False(t, ok)
	})
}

func TestDetectConditionalGateForEach(t *testing.T) {
	t.Parallel()

	t.Run("matches gate", func(t *testing.T) {
		t.Parallel()
		cond, single, ok := detectConditionalGateForEach(parseExprForTest(t, `var.enabled ? [1] : []`))
		require.True(t, ok)
		_, condOK := cond.(*hclsyntax.ScopeTraversalExpr)
		assert.True(t, condOK, "expected traversal for condition, got %T", cond)
		lit, isLit := single.(*hclsyntax.LiteralValueExpr)
		require.True(t, isLit, "expected literal element, got %T", single)
		assert.True(t, lit.Val.AsBigFloat().IsInt())
	})

	t.Run("rejects two-element truthy branch", func(t *testing.T) {
		t.Parallel()
		_, _, ok := detectConditionalGateForEach(parseExprForTest(t, `var.x ? [1, 2] : []`))
		assert.False(t, ok)
	})

	t.Run("rejects non-empty falsy branch", func(t *testing.T) {
		t.Parallel()
		_, _, ok := detectConditionalGateForEach(parseExprForTest(t, `var.x ? [1] : [0]`))
		assert.False(t, ok)
	})

	t.Run("rejects non-tuple branches", func(t *testing.T) {
		t.Parallel()
		_, _, ok := detectConditionalGateForEach(parseExprForTest(t, `var.x ? toset([1]) : toset([])`))
		assert.False(t, ok)
	})

	t.Run("rejects non-conditional expression", func(t *testing.T) {
		t.Parallel()
		_, _, ok := detectConditionalGateForEach(parseExprForTest(t, `var.x`))
		assert.False(t, ok)
	})
}

func TestTosetForEachRangeTokens(t *testing.T) {
	t.Parallel()
	inner := hclwrite.Tokens{
		&hclwrite.Token{Type: hclsyntax.TokenIdent, Bytes: []byte("xs")},
	}
	got := tosetForEachRangeTokens(inner)
	f := hclwrite.NewEmptyFile()
	f.Body().SetAttributeRaw("v", got)
	assert.Equal(t, "v = { for entry in xs : entry => entry }\n", string(hclwrite.Format(f.Bytes())))
}

func TestResolveLatestProviderVersion(t *testing.T) {
	t.Parallel()
	name := impliedProvider("tfe_organization")
	provider, err := resolveRequiredProviderWithRegistrySource(&TestRegistrySource{}, name)
	assert.NoError(t, err)
	assert.Equal(t, "tfe", provider.Name)
	assert.Equal(t, "registry.terraform.io/hashicorp/tfe", provider.Source)
	// latest version resolved which unfortunately can change over time
	// expected that this test may fail in the future alongside the implicit_required_provider
	// test program
	assert.Equal(t, "~> 0.71.0", provider.Requirement.Required.String())
}
