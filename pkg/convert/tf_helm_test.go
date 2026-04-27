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

	t.Run("nested maps deep-merge", func(t *testing.T) {
		t.Parallel()
		// Helm merges maps recursively: keys unique to each side survive, keys
		// present in both take the later document's value. A shallow merge would
		// clobber image.repository here.
		expr := parseExprForTest(t, `[<<EOT
image:
  repository: nginx
  tag: "1.0"
resources:
  limits:
    cpu: "100m"
EOT
, <<EOT
image:
  tag: "2.0"
resources:
  limits:
    memory: "256Mi"
  requests:
    cpu: "50m"
EOT
]`)
		tuple := expr.(*hclsyntax.TupleConsExpr)
		got, ok := mergeStaticHelmValues(tuple)
		require.True(t, ok)

		wantImage := cty.ObjectVal(map[string]cty.Value{
			"repository": cty.StringVal("nginx"),
			"tag":        cty.StringVal("2.0"),
		})
		assert.True(t, got["image"].RawEquals(wantImage),
			"image map should deep-merge; got %#v", got["image"])

		wantResources := cty.ObjectVal(map[string]cty.Value{
			"limits": cty.ObjectVal(map[string]cty.Value{
				"cpu":    cty.StringVal("100m"),
				"memory": cty.StringVal("256Mi"),
			}),
			"requests": cty.ObjectVal(map[string]cty.Value{
				"cpu": cty.StringVal("50m"),
			}),
		})
		assert.True(t, got["resources"].RawEquals(wantResources),
			"resources map should deep-merge recursively; got %#v", got["resources"])
	})

	t.Run("list replaces list (no concat)", func(t *testing.T) {
		t.Parallel()
		// Helm replaces lists wholesale rather than concatenating.
		expr := parseExprForTest(t, `[<<EOT
ports: [80, 443]
EOT
, <<EOT
ports: [8080]
EOT
]`)
		tuple := expr.(*hclsyntax.TupleConsExpr)
		got, ok := mergeStaticHelmValues(tuple)
		require.True(t, ok)
		want := cty.TupleVal([]cty.Value{cty.NumberIntVal(8080)})
		assert.True(t, got["ports"].RawEquals(want), "got %#v", got["ports"])
	})

	t.Run("type mismatch takes later value", func(t *testing.T) {
		t.Parallel()
		// When one doc has a map and the next has a scalar at the same key, the
		// scalar replaces the map (and vice versa).
		expr := parseExprForTest(t, `[<<EOT
image:
  repository: nginx
EOT
, <<EOT
image: "just a string"
EOT
]`)
		tuple := expr.(*hclsyntax.TupleConsExpr)
		got, ok := mergeStaticHelmValues(tuple)
		require.True(t, ok)
		assert.True(t, got["image"].RawEquals(cty.StringVal("just a string")),
			"scalar should replace map; got %#v", got["image"])
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

	t.Run("primitive yaml types", func(t *testing.T) {
		t.Parallel()
		// Exercises the non-string branches of yamlValueToCty that real Helm
		// values typically use: booleans, nulls, floats, and the empty-container
		// fast paths.
		expr := parseExprForTest(t, `[<<EOT
enabled: true
disabled: false
optional:
ratio: 0.5
extras: []
metadata: {}
EOT
]`)
		tuple := expr.(*hclsyntax.TupleConsExpr)
		got, ok := mergeStaticHelmValues(tuple)
		require.True(t, ok)
		assert.True(t, got["enabled"].RawEquals(cty.True))
		assert.True(t, got["disabled"].RawEquals(cty.False))
		assert.True(t, got["optional"].RawEquals(cty.NullVal(cty.DynamicPseudoType)))
		assert.True(t, got["ratio"].RawEquals(cty.NumberFloatVal(0.5)))
		assert.True(t, got["extras"].RawEquals(cty.EmptyTupleVal))
		assert.True(t, got["metadata"].RawEquals(cty.EmptyObjectVal))
	})

	t.Run("yaml type unsupported by cty rejects", func(t *testing.T) {
		t.Parallel()
		// yaml.v3 unmarshals !!timestamp tagged values into time.Time, which
		// yamlValueToCty cannot represent. The whole list should be rejected.
		expr := parseExprForTest(t, `[<<EOT
created: 2026-04-24T10:00:00Z
EOT
]`)
		tuple := expr.(*hclsyntax.TupleConsExpr)
		_, ok := mergeStaticHelmValues(tuple)
		assert.False(t, ok, "time.Time leaves should fall through to the unsupported-type path")
	})
}

// TestYamlValueToCtyUnsupported covers the default branch of yamlValueToCty
// for Go types that yaml.v3 never produces (struct{}, func()) but that the
// function defensively handles. Reachable types are covered end-to-end via
// TestMergeStaticHelmValues.
func TestYamlValueToCtyUnsupported(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		input interface{}
	}{
		{"struct", struct{ X int }{X: 1}},
		{"func", func() {}},
		{"channel", make(chan int)},
		{"list with unsupported element", []interface{}{struct{}{}}},
		{"map with unsupported value", map[string]interface{}{"k": struct{}{}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := yamlValueToCty(tc.input)
			assert.False(t, ok, "unsupported type should return ok=false")
			assert.Equal(t, cty.NilVal, got)
		})
	}

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
