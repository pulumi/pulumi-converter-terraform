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

// Specialized conversion for the Terraform `helm_release` resource. The
// Terraform helm provider exposes shapes that the standard
// GetMapping / convertBody pipeline cannot translate to
// `kubernetes:helm.sh/v3:Release`, namely repeated
// `set{}` / `set_list{}` / `set_sensitive{}` blocks that need to be collapsed
// into a single `values` map and flat `repository_*` attrs that need to be
// re-parented under a synthesized `repositoryOpts` object. Naming and generic
// attribute conversion are still delegated to convertBody (and via it to
// GetMapping, which lets pulumi-kubernetes ship the few field renames it needs).

package convert

import (
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/contract"
	"github.com/zclconf/go-cty/cty"
	yaml "gopkg.in/yaml.v3"
)

// customResourceMapping describes a resource type that needs specialized conversion
// handling in addition to whatever GetMapping provides. The convert function replaces
// the default convertBody call for matching TF resources.
type customResourceMapping struct {
	pulumiToken string
	convert     func(*convertState, *scopes, string, hcl.Body) bodyAttrsTokens
}

// resources that require specialized conversion handling beyond what the standard
// GetMapping / convertBody pipeline can express (e.g. block-shape transforms).
//
//nolint:gosec // G101: these are Pulumi resource tokens, not credentials.
var customResourceMappings = map[string]customResourceMapping{
	"helm_release": {
		pulumiToken: "kubernetes:helm.sh/v3:Release",
		convert:     convertHelmReleaseResource,
	},
}

func isCustomResourceMapping(tfType string) bool {
	_, ok := customResourceMappings[tfType]
	return ok
}

// helm_release blocks whose name/value pairs collapse into a single `values` map.
var helmReleaseSetBlockTypes = map[string]bool{
	"set":           true,
	"set_list":      true,
	"set_sensitive": true,
}

// helm_release flat TF attributes that get restructured under a nested
// `repositoryOpts` object. Values are the nested Pulumi field names.
var helmReleaseRepositoryFields = map[string]string{
	"repository":           "repo",
	"repository_ca_file":   "caFile",
	"repository_cert_file": "certFile",
	"repository_key_file":  "keyFile",
	"repository_username":  "username",
	"repository_password":  "password",
}

// mergeStaticHelmValues parses each element of a TF `values = [...]` list as a
// YAML document and deep-merges them left-to-right (later wins, matching Helm's
// values merge semantics: map+map recurses, everything else replaces). Returns
// (merged, true) only when every list element is a static string that
// yaml-unmarshals and every leaf value maps to a cty type we can represent in
// PCL (strings, numbers, bools, maps, lists).
func mergeStaticHelmValues(tupleExpr *hclsyntax.TupleConsExpr) (map[string]cty.Value, bool) {
	merged := map[string]interface{}{}
	for _, item := range tupleExpr.Exprs {
		str, _ := matchStaticString(item)
		if str == nil {
			return nil, false
		}
		var parsed map[string]interface{}
		if err := yaml.Unmarshal([]byte(*str), &parsed); err != nil {
			return nil, false
		}
		deepMergeHelmValues(merged, parsed)
	}
	out := make(map[string]cty.Value, len(merged))
	for k, v := range merged {
		cv, ok := yamlValueToCty(v)
		if !ok {
			return nil, false
		}
		out[k] = cv
	}
	return out, true
}

// deepMergeHelmValues merges src into dst, matching Helm's values-merge rules:
// when both sides hold a map at the same key, recurse into it; otherwise the src
// value replaces dst's (including lists — Helm does not concatenate them, and
// type mismatches always take the later value).
func deepMergeHelmValues(dst, src map[string]interface{}) {
	for k, srcV := range src {
		if dstV, seen := dst[k]; seen {
			dstMap, dstIsMap := dstV.(map[string]interface{})
			srcMap, srcIsMap := srcV.(map[string]interface{})
			if dstIsMap && srcIsMap {
				deepMergeHelmValues(dstMap, srcMap)
				continue
			}
		}
		dst[k] = srcV
	}
}

// yamlValueToCty recursively converts a value produced by yaml.Unmarshal into
// interface{} into the equivalent cty.Value. Returns (_, false) if it encounters
// a type hclwrite can't emit.
func yamlValueToCty(v interface{}) (cty.Value, bool) {
	switch val := v.(type) {
	case nil:
		return cty.NullVal(cty.DynamicPseudoType), true
	case string:
		return cty.StringVal(val), true
	case bool:
		return cty.BoolVal(val), true
	case int:
		return cty.NumberIntVal(int64(val)), true
	case int64:
		return cty.NumberIntVal(val), true
	case uint64:
		return cty.NumberUIntVal(val), true
	case float64:
		return cty.NumberFloatVal(val), true
	case []interface{}:
		if len(val) == 0 {
			return cty.EmptyTupleVal, true
		}
		items := make([]cty.Value, 0, len(val))
		for _, item := range val {
			cv, ok := yamlValueToCty(item)
			if !ok {
				return cty.NilVal, false
			}
			items = append(items, cv)
		}
		return cty.TupleVal(items), true
	case map[string]interface{}:
		if len(val) == 0 {
			return cty.EmptyObjectVal, true
		}
		attrs := make(map[string]cty.Value, len(val))
		for k, mv := range val {
			cv, ok := yamlValueToCty(mv)
			if !ok {
				return cty.NilVal, false
			}
			attrs[k] = cv
		}
		return cty.ObjectVal(attrs), true
	}
	return cty.NilVal, false
}

// staticPostrenderCommand flattens a TF `postrender { binary_path, args }` block
// into the single Pulumi postrender command string. Returns (cmd, true) only when
// binary_path and every args element are static strings.
func staticPostrenderCommand(block *hclsyntax.Block) (string, bool) {
	inner := bodyContent(block.Body)
	binPathAttr, hasBin := inner.Attributes["binary_path"]
	if !hasBin {
		return "", false
	}
	binExpr, ok := binPathAttr.Expr.(hclsyntax.Expression)
	if !ok {
		return "", false
	}
	binPath, _ := matchStaticString(binExpr)
	if binPath == nil {
		return "", false
	}
	parts := []string{*binPath}
	if argsAttr, hasArgs := inner.Attributes["args"]; hasArgs {
		tuple, ok := argsAttr.Expr.(*hclsyntax.TupleConsExpr)
		if !ok {
			return "", false
		}
		for _, item := range tuple.Exprs {
			s, _ := matchStaticString(item)
			if s == nil {
				return "", false
			}
			parts = append(parts, *s)
		}
	}
	return strings.Join(parts, " "), true
}

// convertHelmReleaseResource handles the shape transforms that GetMapping cannot
// express for helm_release → kubernetes:helm.sh/v3:Release: collapsing
// set/set_list/set_sensitive blocks into a single `values` map (wrapping
// set_sensitive with secret()) and collecting flat repository_* attributes into a
// nested repositoryOpts object. Naming and generic attribute conversion are
// delegated to convertBody, which uses the helm GetMapping to find the right
// Pulumi names.
func convertHelmReleaseResource(
	state *convertState, scopes *scopes, fullyQualifiedPath string, body hcl.Body,
) bodyAttrsTokens {
	contract.Assertf(fullyQualifiedPath != "", "fullyQualifiedPath should not be empty")

	synbody, ok := body.(*hclsyntax.Body)
	contract.Assertf(ok, "%T was not a hclsyntax.Body", body)

	// Collect set* blocks into a synthesized `values` object and postrender blocks
	// into a synthesized `postrender` string command.
	valueAttrs := make([]hclwrite.ObjectAttrTokens, 0)
	firstValueLine := 0
	postrenderCmd := ""
	postrenderLine := 0
	// Names set via set/set_list/set_sensitive blocks (when the name is a static
	// string). Used to dedupe against YAML keys from the values = [...] list —
	// set-block overrides win per Helm semantics.
	setBlockStaticNames := make(map[string]bool)
	filteredBlocks := make(hclsyntax.Blocks, 0, len(synbody.Blocks))
	for _, block := range synbody.Blocks {
		if block.Type == "postrender" {
			cmd, ok := staticPostrenderCommand(block)
			if !ok {
				state.appendDiagnostic(&hcl.Diagnostic{
					Severity: hcl.DiagWarning,
					Subject:  block.DefRange().Ptr(),
					Summary:  "postrender block not translated",
					Detail: "kubernetes.helm.v3.Release.postrender is a single command string. " +
						"The Terraform postrender block was dropped because binary_path or args " +
						"is not a static string. Set postrender manually on the converted resource.",
				})
				continue
			}
			postrenderCmd = cmd
			postrenderLine = block.DefRange().Start.Line
			continue
		}
		if !helmReleaseSetBlockTypes[block.Type] {
			filteredBlocks = append(filteredBlocks, block)
			continue
		}
		blockPath := appendPath(fullyQualifiedPath, block.Type)
		inner := bodyContent(block.Body)
		nameAttr, hasName := inner.Attributes["name"]
		valueAttr, hasValue := inner.Attributes["value"]
		if !hasName || !hasValue {
			continue
		}
		valueTokens := convertExpression(state, true, scopes, blockPath, valueAttr.Expr)
		if block.Type == "set_sensitive" {
			valueTokens = hclwrite.TokensForFunctionCall("secret", valueTokens)
		}
		if firstValueLine == 0 {
			firstValueLine = valueAttr.Range.Start.Line
		}
		if nameExpr, ok := nameAttr.Expr.(hclsyntax.Expression); ok {
			if nameStr, _ := matchStaticString(nameExpr); nameStr != nil {
				setBlockStaticNames[*nameStr] = true
			}
		}
		valueAttrs = append(valueAttrs, hclwrite.ObjectAttrTokens{
			Name:  convertExpression(state, true, scopes, blockPath, nameAttr.Expr),
			Value: valueTokens,
		})
	}

	// Collect repository_* attributes into a synthesized `repositoryOpts` object,
	// and pull `wait` aside so we can emit it as the inverse Pulumi `skipAwait`.
	repoAttrs := make([]bodyAttrTokens, 0)
	filteredAttrs := make(hclsyntax.Attributes, len(synbody.Attributes))
	var waitAttr *hclsyntax.Attribute
	for name, attr := range synbody.Attributes {
		if name == "wait" {
			waitAttr = attr
			continue
		}
		if name == "pass_credentials" {
			nameRange := attr.NameRange
			state.appendDiagnostic(&hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Subject:  &nameRange,
				Summary:  "pass_credentials not supported",
				Detail: "kubernetes.helm.v3.Release has no pass_credentials equivalent; " +
					"the Terraform attribute was dropped.",
			})
			continue
		}
		nested, isRepo := helmReleaseRepositoryFields[name]
		if !isRepo {
			filteredAttrs[name] = attr
			continue
		}
		attrPath := appendPath(fullyQualifiedPath, name)
		leading, _ := getTrivia(state.sources, getAttributeRange(state.sources, attr.Expr.Range()), true)
		repoAttrs = append(repoAttrs, bodyAttrTokens{
			Line:   attr.Range().Start.Line,
			Name:   nested,
			Trivia: leading,
			Value:  convertExpression(state, true, scopes, attrPath, attr.Expr),
		})
	}

	// Merge the TF `values = [<<YAML...]` list into the Pulumi values map. TF's
	// values is a list of YAML documents that Helm parses and deep-merges at
	// apply time; Pulumi's values is an already-parsed map. We parse each YAML
	// document statically, merge them (later overrides earlier), and emit the
	// keys into valueAttrs (skipping any keys a set-block already sets — set
	// blocks win per Helm). YAML entries are prepended so the set-block
	// overrides read below them in the output, matching the TF layering.
	if yamlAttr, hasYaml := filteredAttrs["values"]; hasYaml {
		yamlRange := yamlAttr.SrcRange
		tupleExpr, isTuple := yamlAttr.Expr.(*hclsyntax.TupleConsExpr)
		switch {
		case !isTuple:
			state.appendDiagnostic(&hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Subject:  &yamlRange,
				Summary:  "values list not a static tuple",
				Detail: "Expected a list literal for helm_release values; the " +
					"attribute was dropped. Reconstruct it manually as a Pulumi " +
					"values map on the converted resource.",
			})
		default:
			merged, ok := mergeStaticHelmValues(tupleExpr)
			if !ok {
				state.appendDiagnostic(&hcl.Diagnostic{
					Severity: hcl.DiagWarning,
					Subject:  &yamlRange,
					Summary:  "values YAML not statically parseable",
					Detail: "Each element of helm_release values = [...] must be " +
						"a static YAML document whose leaves are strings, numbers, " +
						"booleans, maps, or lists. The attribute was dropped.",
				})
			} else {
				keys := make([]string, 0, len(merged))
				for k := range merged {
					if setBlockStaticNames[k] {
						continue
					}
					keys = append(keys, k)
				}
				sort.Strings(keys)
				yamlEntries := make([]hclwrite.ObjectAttrTokens, 0, len(keys))
				for _, k := range keys {
					yamlEntries = append(yamlEntries, hclwrite.ObjectAttrTokens{
						Name:  hclwrite.TokensForValue(cty.StringVal(k)),
						Value: hclwrite.TokensForValue(merged[k]),
					})
				}
				valueAttrs = append(yamlEntries, valueAttrs...)
				if firstValueLine == 0 {
					firstValueLine = yamlRange.Start.Line
				}
			}
		}
		delete(filteredAttrs, "values")
	}

	sort.Slice(repoAttrs, func(i, j int) bool { return repoAttrs[i].Line < repoAttrs[j].Line })

	filteredBody := &hclsyntax.Body{
		Attributes: filteredAttrs,
		Blocks:     filteredBlocks,
		SrcRange:   synbody.SrcRange,
		EndRange:   synbody.EndRange,
	}
	result := convertBody(state, scopes, fullyQualifiedPath, filteredBody)

	if len(repoAttrs) > 0 {
		result = append(result, bodyAttrTokens{
			Line:   repoAttrs[0].Line,
			Name:   "repositoryOpts",
			Trivia: make(hclwrite.Tokens, 0),
			Value:  tokensForObject(repoAttrs),
		})
	}
	if len(valueAttrs) > 0 {
		result = append(result, bodyAttrTokens{
			Line:   firstValueLine,
			Name:   "values",
			Trivia: make(hclwrite.Tokens, 0),
			Value:  hclwrite.TokensForObject(valueAttrs),
		})
	}
	if postrenderCmd != "" {
		result = append(result, bodyAttrTokens{
			Line:   postrenderLine,
			Name:   "postrender",
			Trivia: make(hclwrite.Tokens, 0),
			Value:  hclwrite.TokensForValue(cty.StringVal(postrenderCmd)),
		})
	}
	// TF `wait` (default true) is the inverse of Pulumi `skipAwait` (default false).
	if waitAttr != nil {
		attrPath := appendPath(fullyQualifiedPath, "wait")
		leading, _ := getTrivia(state.sources, getAttributeRange(state.sources, waitAttr.Expr.Range()), true)
		waitExpr := convertExpression(state, true, scopes, attrPath, waitAttr.Expr)
		skipTokens := make(hclwrite.Tokens, 0, 1+len(waitExpr))
		skipTokens = append(skipTokens, makeToken(hclsyntax.TokenBang, "!"))
		skipTokens = append(skipTokens, waitExpr...)
		result = append(result, bodyAttrTokens{
			Line:   waitAttr.Range().Start.Line,
			Name:   "skipAwait",
			Trivia: leading,
			Value:  skipTokens,
		})
	}
	sort.Sort(result)
	return result
}
