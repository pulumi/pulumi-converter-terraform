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

// parseTupleForTest parses an HCL tuple-literal expression like `[<<EOT...EOT]`
// and returns it typed for direct use with mergeStaticHelmValues.
func parseTupleForTest(t *testing.T, src string) *hclsyntax.TupleConsExpr {
	t.Helper()
	expr, diags := hclsyntax.ParseExpression([]byte(src), "test.hcl", hcl.Pos{Line: 1, Column: 1})
	require.False(t, diags.HasErrors(), "parse failed: %s", diags)
	tuple, ok := expr.(*hclsyntax.TupleConsExpr)
	require.True(t, ok, "expected a tuple expression, got %T", expr)
	return tuple
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
		got, ok := mergeStaticHelmValues(parseTupleForTest(t, `[<<EOT
a: 1
b: two
EOT
]`))
		require.True(t, ok)
		assert.Equal(t, cty.NumberIntVal(1), got["a"])
		assert.Equal(t, cty.StringVal("two"), got["b"])
	})

	t.Run("multi-document later wins", func(t *testing.T) {
		t.Parallel()
		got, ok := mergeStaticHelmValues(parseTupleForTest(t, `[<<EOT
shared: first
only_first: keep
EOT
, <<EOT
shared: second
only_second: keep
EOT
]`))
		require.True(t, ok)
		assert.Equal(t, cty.StringVal("second"), got["shared"])
		assert.Equal(t, cty.StringVal("keep"), got["only_first"])
		assert.Equal(t, cty.StringVal("keep"), got["only_second"])
	})

	t.Run("nested maps deep-merge", func(t *testing.T) {
		t.Parallel()
		// Helm merges maps recursively; a shallow merge would clobber image.repository.
		got, ok := mergeStaticHelmValues(parseTupleForTest(t, `[<<EOT
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
]`))
		require.True(t, ok)
		assert.Equal(t, cty.ObjectVal(map[string]cty.Value{
			"repository": cty.StringVal("nginx"),
			"tag":        cty.StringVal("2.0"),
		}), got["image"])
		assert.Equal(t, cty.ObjectVal(map[string]cty.Value{
			"limits": cty.ObjectVal(map[string]cty.Value{
				"cpu":    cty.StringVal("100m"),
				"memory": cty.StringVal("256Mi"),
			}),
			"requests": cty.ObjectVal(map[string]cty.Value{
				"cpu": cty.StringVal("50m"),
			}),
		}), got["resources"])
	})

	t.Run("list replaces list (no concat)", func(t *testing.T) {
		t.Parallel()
		// Helm replaces lists wholesale rather than concatenating.
		got, ok := mergeStaticHelmValues(parseTupleForTest(t, `[<<EOT
ports: [80, 443]
EOT
, <<EOT
ports: [8080]
EOT
]`))
		require.True(t, ok)
		assert.Equal(t, cty.TupleVal([]cty.Value{cty.NumberIntVal(8080)}), got["ports"])
	})

	t.Run("type mismatch takes later value", func(t *testing.T) {
		t.Parallel()
		got, ok := mergeStaticHelmValues(parseTupleForTest(t, `[<<EOT
image:
  repository: nginx
EOT
, <<EOT
image: "just a string"
EOT
]`))
		require.True(t, ok)
		assert.Equal(t, cty.StringVal("just a string"), got["image"])
	})

	t.Run("nested structures", func(t *testing.T) {
		t.Parallel()
		got, ok := mergeStaticHelmValues(parseTupleForTest(t, `[<<EOT
image:
  repository: nginx
  tag: "1.25"
ports: [80, 443]
EOT
]`))
		require.True(t, ok)
		assert.Equal(t, cty.ObjectVal(map[string]cty.Value{
			"repository": cty.StringVal("nginx"),
			"tag":        cty.StringVal("1.25"),
		}), got["image"])
		assert.Equal(t, cty.TupleVal([]cty.Value{
			cty.NumberIntVal(80),
			cty.NumberIntVal(443),
		}), got["ports"])
	})

	t.Run("empty list", func(t *testing.T) {
		t.Parallel()
		got, ok := mergeStaticHelmValues(parseTupleForTest(t, `[]`))
		require.True(t, ok)
		assert.Empty(t, got)
	})

	t.Run("primitive yaml types", func(t *testing.T) {
		t.Parallel()
		got, ok := mergeStaticHelmValues(parseTupleForTest(t, `[<<EOT
enabled: true
disabled: false
optional:
ratio: 0.5
extras: []
metadata: {}
EOT
]`))
		require.True(t, ok)
		assert.Equal(t, cty.True, got["enabled"])
		assert.Equal(t, cty.False, got["disabled"])
		assert.Equal(t, cty.NullVal(cty.DynamicPseudoType), got["optional"])
		assert.Equal(t, cty.NumberFloatVal(0.5), got["ratio"])
		assert.Equal(t, cty.EmptyTupleVal, got["extras"])
		assert.Equal(t, cty.EmptyObjectVal, got["metadata"])
	})

	t.Run("yaml type unsupported by cty rejects", func(t *testing.T) {
		t.Parallel()
		// yaml.v3 unmarshals !!timestamp values into time.Time, which yamlValueToCty cannot represent.
		_, ok := mergeStaticHelmValues(parseTupleForTest(t, `[<<EOT
created: 2026-04-24T10:00:00Z
EOT
]`))
		assert.False(t, ok)
	})

	t.Run("dynamic element rejected", func(t *testing.T) {
		t.Parallel()
		_, ok := mergeStaticHelmValues(parseTupleForTest(t, `[var.yaml]`))
		assert.False(t, ok)
	})

	t.Run("malformed YAML rejected", func(t *testing.T) {
		t.Parallel()
		_, ok := mergeStaticHelmValues(parseTupleForTest(t, `["a: b: c: nope: :"]`))
		assert.False(t, ok)
	})

	t.Run("mixed static and dynamic rejected", func(t *testing.T) {
		t.Parallel()
		// Any dynamic element poisons the whole list.
		_, ok := mergeStaticHelmValues(parseTupleForTest(t, `[<<EOT
ok: yes
EOT
, var.yaml]`))
		assert.False(t, ok)
	})
}

// TestYamlValueToCtyUnsupported covers the default branch for Go types yaml.v3
// never produces (struct{}, func()) but that the function defensively handles.
// Reachable types are covered end-to-end via TestMergeStaticHelmValues.
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
			assert.False(t, ok)
			assert.Equal(t, cty.NilVal, got)
		})
	}
}

func TestStaticPostrenderCommand(t *testing.T) {
	t.Parallel()

	t.Run("binary and args static", func(t *testing.T) {
		t.Parallel()
		cmd, ok := staticPostrenderCommand(parseBlockForTest(t, `postrender {
  binary_path = "echo"
  args = [ "foo", "bar" ]
}`))
		require.True(t, ok)
		assert.Equal(t, "echo foo bar", cmd)
	})

	t.Run("binary only no args", func(t *testing.T) {
		t.Parallel()
		cmd, ok := staticPostrenderCommand(parseBlockForTest(t, `postrender {
  binary_path = "my-renderer"
}`))
		require.True(t, ok)
		assert.Equal(t, "my-renderer", cmd)
	})

	t.Run("binary empty args list", func(t *testing.T) {
		t.Parallel()
		cmd, ok := staticPostrenderCommand(parseBlockForTest(t, `postrender {
  binary_path = "cmd"
  args = []
}`))
		require.True(t, ok)
		assert.Equal(t, "cmd", cmd)
	})

	t.Run("missing binary_path rejected", func(t *testing.T) {
		t.Parallel()
		_, ok := staticPostrenderCommand(parseBlockForTest(t, `postrender {
  args = [ "foo" ]
}`))
		assert.False(t, ok)
	})

	t.Run("dynamic binary_path rejected", func(t *testing.T) {
		t.Parallel()
		_, ok := staticPostrenderCommand(parseBlockForTest(t, `postrender {
  binary_path = var.cmd
  args = [ "foo" ]
}`))
		assert.False(t, ok)
	})

	t.Run("dynamic arg rejected", func(t *testing.T) {
		t.Parallel()
		// Any dynamic arg poisons the whole block.
		_, ok := staticPostrenderCommand(parseBlockForTest(t, `postrender {
  binary_path = "echo"
  args = [ var.arg ]
}`))
		assert.False(t, ok)
	})

	t.Run("args not a tuple rejected", func(t *testing.T) {
		t.Parallel()
		_, ok := staticPostrenderCommand(parseBlockForTest(t, `postrender {
  binary_path = "echo"
  args = var.args
}`))
		assert.False(t, ok)
	})
}
