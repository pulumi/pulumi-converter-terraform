// Copyright 2016-2026, Pulumi Corporation.
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

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

// parseExprForTest parses a single HCL expression. Helpers in this file use it
// to construct *hclsyntax.Expression values for testing the helm preprocessor
// primitives.
func parseExprForTest(t *testing.T, src string) hclsyntax.Expression {
	t.Helper()
	expr, diags := hclsyntax.ParseExpression([]byte(src), "test.hcl", hcl.Pos{Line: 1, Column: 1})
	require.False(t, diags.HasErrors(), "parse failed: %s", diags)
	return expr
}

// parseBlockForTest parses `<blockType> { ... }` and returns the first block.
func parseBlockForTest(t *testing.T, src string) *hclsyntax.Block {
	t.Helper()
	file, diags := hclsyntax.ParseConfig([]byte(src), "test.hcl", hcl.Pos{Line: 1, Column: 1})
	require.False(t, diags.HasErrors(), "parse failed: %s", diags)
	body := file.Body.(*hclsyntax.Body)
	require.Len(t, body.Blocks, 1, "expected exactly one block")
	return body.Blocks[0]
}

func TestYamlValueToCty(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input interface{}
		want  cty.Value
		okAs  bool
	}{
		{"nil", nil, cty.NullVal(cty.DynamicPseudoType), true},
		{"string", "hello", cty.StringVal("hello"), true},
		{"empty string", "", cty.StringVal(""), true},
		{"bool true", true, cty.True, true},
		{"bool false", false, cty.False, true},
		{"int", 42, cty.NumberIntVal(42), true},
		{"int64", int64(7), cty.NumberIntVal(7), true},
		{"uint64", uint64(9), cty.NumberUIntVal(9), true},
		{"float", 3.14, cty.NumberFloatVal(3.14), true},

		{"empty list", []interface{}{}, cty.EmptyTupleVal, true},
		{"list of strings", []interface{}{"a", "b"},
			cty.TupleVal([]cty.Value{cty.StringVal("a"), cty.StringVal("b")}), true},
		{"mixed list", []interface{}{"s", 1, true},
			cty.TupleVal([]cty.Value{cty.StringVal("s"), cty.NumberIntVal(1), cty.True}), true},

		{"empty map", map[string]interface{}{}, cty.EmptyObjectVal, true},
		{"flat map",
			map[string]interface{}{"k": "v"},
			cty.ObjectVal(map[string]cty.Value{"k": cty.StringVal("v")}),
			true,
		},
		{"nested map",
			map[string]interface{}{
				"a": map[string]interface{}{"b": int(1)},
			},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.ObjectVal(map[string]cty.Value{
					"b": cty.NumberIntVal(1),
				}),
			}),
			true,
		},
		{"list of maps",
			[]interface{}{
				map[string]interface{}{"x": 1},
				map[string]interface{}{"y": 2},
			},
			cty.TupleVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{"x": cty.NumberIntVal(1)}),
				cty.ObjectVal(map[string]cty.Value{"y": cty.NumberIntVal(2)}),
			}),
			true,
		},

		{"unsupported struct", struct{ X int }{X: 1}, cty.NilVal, false},
		{"unsupported func", func() {}, cty.NilVal, false},
		{"list with unsupported", []interface{}{struct{}{}}, cty.NilVal, false},
		{"map with unsupported", map[string]interface{}{"k": struct{}{}}, cty.NilVal, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := yamlValueToCty(tc.input)
			assert.Equal(t, tc.okAs, ok, "ok flag")
			if !tc.okAs {
				return
			}
			assert.True(t, got.RawEquals(tc.want), "want %#v, got %#v", tc.want, got)
		})
	}
}

func TestMergeStaticHelmValues(t *testing.T) {
	t.Parallel()

	t.Run("single document", func(t *testing.T) {
		t.Parallel()
		expr := parseExprForTest(t, `[<<EOT
a: 1
b: two
EOT
]`)
		tuple := expr.(*hclsyntax.TupleConsExpr)
		got, ok := mergeStaticHelmValues(tuple)
		require.True(t, ok)
		assert.True(t, got["a"].RawEquals(cty.NumberIntVal(1)))
		assert.True(t, got["b"].RawEquals(cty.StringVal("two")))
	})

	t.Run("multi-document later wins", func(t *testing.T) {
		t.Parallel()
		expr := parseExprForTest(t, `[<<EOT
shared: first
only_first: keep
EOT
, <<EOT
shared: second
only_second: keep
EOT
]`)
		tuple := expr.(*hclsyntax.TupleConsExpr)
		got, ok := mergeStaticHelmValues(tuple)
		require.True(t, ok)
		assert.True(t, got["shared"].RawEquals(cty.StringVal("second")), "later document wins")
		assert.True(t, got["only_first"].RawEquals(cty.StringVal("keep")))
		assert.True(t, got["only_second"].RawEquals(cty.StringVal("keep")))
	})

	t.Run("nested structures", func(t *testing.T) {
		t.Parallel()
		expr := parseExprForTest(t, `[<<EOT
image:
  repository: nginx
  tag: "1.25"
ports: [80, 443]
EOT
]`)
		tuple := expr.(*hclsyntax.TupleConsExpr)
		got, ok := mergeStaticHelmValues(tuple)
		require.True(t, ok)
		wantImage := cty.ObjectVal(map[string]cty.Value{
			"repository": cty.StringVal("nginx"),
			"tag":        cty.StringVal("1.25"),
		})
		assert.True(t, got["image"].RawEquals(wantImage))
		wantPorts := cty.TupleVal([]cty.Value{
			cty.NumberIntVal(80),
			cty.NumberIntVal(443),
		})
		assert.True(t, got["ports"].RawEquals(wantPorts))
	})

	t.Run("empty list", func(t *testing.T) {
		t.Parallel()
		expr := parseExprForTest(t, `[]`)
		tuple := expr.(*hclsyntax.TupleConsExpr)
		got, ok := mergeStaticHelmValues(tuple)
		require.True(t, ok)
		assert.Empty(t, got)
	})

	t.Run("dynamic element rejected", func(t *testing.T) {
		t.Parallel()
		expr := parseExprForTest(t, `[var.yaml]`)
		tuple := expr.(*hclsyntax.TupleConsExpr)
		_, ok := mergeStaticHelmValues(tuple)
		assert.False(t, ok, "variable reference in values should not parse statically")
	})

	t.Run("malformed YAML rejected", func(t *testing.T) {
		t.Parallel()
		expr := parseExprForTest(t, `["a: b: c: nope: :"]`)
		tuple := expr.(*hclsyntax.TupleConsExpr)
		_, ok := mergeStaticHelmValues(tuple)
		assert.False(t, ok, "malformed YAML should be rejected")
	})

	t.Run("mixed static and dynamic rejected", func(t *testing.T) {
		t.Parallel()
		expr := parseExprForTest(t, `[<<EOT
ok: yes
EOT
, var.yaml]`)
		tuple := expr.(*hclsyntax.TupleConsExpr)
		_, ok := mergeStaticHelmValues(tuple)
		assert.False(t, ok, "any dynamic element poisons the whole list")
	})
}

func TestStaticPostrenderCommand(t *testing.T) {
	t.Parallel()

	t.Run("binary and args static", func(t *testing.T) {
		t.Parallel()
		block := parseBlockForTest(t, `postrender {
  binary_path = "echo"
  args = [ "foo", "bar" ]
}`)
		cmd, ok := staticPostrenderCommand(block)
		require.True(t, ok)
		assert.Equal(t, "echo foo bar", cmd)
	})

	t.Run("binary only no args", func(t *testing.T) {
		t.Parallel()
		block := parseBlockForTest(t, `postrender {
  binary_path = "my-renderer"
}`)
		cmd, ok := staticPostrenderCommand(block)
		require.True(t, ok)
		assert.Equal(t, "my-renderer", cmd)
	})

	t.Run("binary empty args list", func(t *testing.T) {
		t.Parallel()
		block := parseBlockForTest(t, `postrender {
  binary_path = "cmd"
  args = []
}`)
		cmd, ok := staticPostrenderCommand(block)
		require.True(t, ok)
		assert.Equal(t, "cmd", cmd)
	})

	t.Run("missing binary_path rejected", func(t *testing.T) {
		t.Parallel()
		block := parseBlockForTest(t, `postrender {
  args = [ "foo" ]
}`)
		_, ok := staticPostrenderCommand(block)
		assert.False(t, ok, "binary_path is required")
	})

	t.Run("dynamic binary_path rejected", func(t *testing.T) {
		t.Parallel()
		block := parseBlockForTest(t, `postrender {
  binary_path = var.cmd
  args = [ "foo" ]
}`)
		_, ok := staticPostrenderCommand(block)
		assert.False(t, ok, "dynamic binary_path should not extract statically")
	})

	t.Run("dynamic arg rejected", func(t *testing.T) {
		t.Parallel()
		block := parseBlockForTest(t, `postrender {
  binary_path = "echo"
  args = [ var.arg ]
}`)
		_, ok := staticPostrenderCommand(block)
		assert.False(t, ok, "any dynamic arg poisons the whole block")
	})

	t.Run("args not a tuple rejected", func(t *testing.T) {
		t.Parallel()
		block := parseBlockForTest(t, `postrender {
  binary_path = "echo"
  args = var.args
}`)
		_, ok := staticPostrenderCommand(block)
		assert.False(t, ok, "args must be a tuple literal")
	})
}
