// Copyright 2016-2025, Pulumi Corporation.
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
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/apparentlymart/go-versions/versions"
	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform-svchost/disco"
	"github.com/opentofu/opentofu/shim"
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfbridge"
	"github.com/pulumi/pulumi/pkg/v3/codegen"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/contract"
	"github.com/pulumi/pulumi/sdk/v3/go/common/workspace"
	"github.com/pulumi/terraform/pkg/addrs"
	"github.com/pulumi/terraform/pkg/configs"
	"github.com/pulumi/terraform/pkg/getmodules"
	"github.com/pulumi/terraform/pkg/registry"
	"github.com/pulumi/terraform/pkg/registry/regsrc"
	"github.com/spf13/afero"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
	"golang.org/x/exp/maps"

	yaml "gopkg.in/yaml.v3"
)

const pulumiTerraformModuleAnnotation = "@pulumi-terraform-module"

type PulumiTerraformModule struct {
	packageName     string
	moduleCall      *configs.ModuleCall
	local           bool
	absolutePath    string
	targetDirectory string
	copyModule      bool
}

func changeExtension(path, ext string) string {
	dir, file := filepath.Split(path)
	dotIndex := strings.LastIndex(file, ".")
	if dotIndex == -1 {
		return dir + file
	}

	base := file[:dotIndex]
	return dir + base + ext
}

func loadConfigDir(fs afero.Fs, path string) (map[string][]byte, *configs.Module, hcl.Diagnostics) {
	p := configs.NewParser(fs)
	mod, diags := p.LoadConfigDir(path)
	return p.Sources(), mod, diags
}

func inferPrimitiveType(input cty.Type, defaultType string) string {
	if input.Equals(cty.Number) {
		return "number"
	}
	if input.Equals(cty.Bool) {
		return "bool"
	}
	if input.Equals(cty.String) {
		return "string"
	}

	return defaultType
}

func convertCtyType(typ cty.Type) string {
	if typ.Equals(cty.Number) {
		return "number"
	}
	if typ.Equals(cty.Bool) {
		return "bool"
	}
	if typ.Equals(cty.String) {
		return "string"
	}
	if typ.IsListType() {
		elementType := convertCtyType(typ.ElementType())
		return fmt.Sprintf("list(%s)", elementType)
	}
	if typ.IsMapType() {
		elementType := convertCtyType(typ.ElementType())
		return fmt.Sprintf("map(%s)", elementType)
	}
	if typ.IsSetType() {
		// handle sets like lists
		elementType := convertCtyType(typ.ElementType())
		return fmt.Sprintf("list(%s)", elementType)
	}
	if typ.IsObjectType() {
		attributeKeys := []string{}
		for attributeKey := range typ.AttributeTypes() {
			attributeKeys = append(attributeKeys, attributeKey)
		}

		// sort the attribute keys so that the resulting object type (as text) is deterministic
		sort.Strings(attributeKeys)
		attributeTypes := typ.AttributeTypes()
		attributes := []string{}
		for _, attributeKey := range attributeKeys {
			attributeType := attributeTypes[attributeKey]
			// rename the attribute keys to match pulumi style (camelCase)
			attributeKey = camelCaseName(attributeKey)
			attributes = append(attributes, fmt.Sprintf("%s=%s", attributeKey, convertCtyType(attributeType)))
		}

		attributePairs := ""
		length := len(attributes)
		for i, attribute := range attributes {
			attributePairs = attributePairs + attribute
			if i < length-1 {
				// add a comma to all pairs but the last one
				attributePairs = attributePairs + ", "
			}
		}

		if len(attributes) == 0 {
			// empty object, treat it as dynamic
			return "any"
		}

		return fmt.Sprintf("object({%s})", attributePairs)
	}

	// If we got here it's probably the "dynamic type" and we just report back "any"
	return "any"
}

// Returns true if the token type is trivia (a comment or new line)
func isTrivia(tokenType hclsyntax.TokenType) bool {
	return tokenType == hclsyntax.TokenComment || tokenType == hclsyntax.TokenNewline
}

// Return the trivia before first token index, and after the last token index. Note that last is inclusive and
// may be equal to first. blockLike toggles the search to treat new lines and brackets as semantically
// significant, see the doc comment on getTrivia for more details.
func getTrivaFromIndex(tokens hclsyntax.Tokens, first, last int, blockLike bool) (hclwrite.Tokens, hclwrite.Tokens) {
	contract.Assertf(first <= last, "first (%d) must be <= last (%d)", first, last)

	// Work backwards from first to build up leading trivia
	leading := make(hclwrite.Tokens, 0)
	first = first - 1
	newlineIndex := -1
	hitBrace := false
	for first >= 0 {
		if tokens[first].Type == hclsyntax.TokenNewline {
			newlineIndex = len(leading)
		}
		if isTrivia(tokens[first].Type) {
			leading = append(leading, &hclwrite.Token{
				Type:  tokens[first].Type,
				Bytes: tokens[first].Bytes,
			})
			first = first - 1
		} else {
			hitBrace = tokens[first].Type == hclsyntax.TokenOBrace
			break
		}
	}

	// If we're not in block mode, or we are in block mode but we hit a brace then we're taking all the
	// leading trivia. But otherwise we need to cut the first line of trivia because it will be associated
	// with the item before us in the block.
	if blockLike && !hitBrace && newlineIndex != -1 {
		leading = leading[0:newlineIndex]
	}

	// Drop the first trailing new line if any, we'll add it back later building up attributes and blocks
	if len(leading) > 0 && leading[0].Type == hclsyntax.TokenNewline {
		leading = leading[1:]
	}
	// Reverse the list, we built leading up backwards
	for i, j := 0, len(leading)-1; i < j; i, j = i+1, j-1 {
		leading[i], leading[j] = leading[j], leading[i]
	}

	// Now work forwards from last to build up trailing trivia
	trailing := make(hclwrite.Tokens, 0)
	last = last + 1
	newlineIndex = -1
	hitBrace = false
	for last < len(tokens) {
		if newlineIndex == -1 && tokens[last].Type == hclsyntax.TokenNewline {
			newlineIndex = len(trailing)
		}
		if isTrivia(tokens[last].Type) {
			trailing = append(trailing, &hclwrite.Token{
				Type:  tokens[last].Type,
				Bytes: tokens[last].Bytes,
			})
			last = last + 1
		} else {
			hitBrace = tokens[last].Type == hclsyntax.TokenCBrace
			break
		}
	}

	// If we're not in block mode, or we are in block mode but we hit a brace then we're taking all the
	// trailing trivia. But otherwise we need to cut after the first new line, everything else will be
	// associated with the next item in the block.
	if blockLike && !hitBrace && newlineIndex != -1 {
		trailing = trailing[0:newlineIndex]
	}

	// Drop the last trailing new line if any, we'll add it back later building up attributes and blocks
	if len(trailing) > 0 && trailing[len(trailing)-1].Type == hclsyntax.TokenNewline {
		trailing = trailing[0 : len(trailing)-1]
	}

	return leading, trailing
}

// Given a HCL range find the trivia before and after that range. Be careful about doubly counting trivia with
// this function. For example, if you have a binary expression `/* leading */ 1 + 2` and you call this
// function for the binary expressions range you'll get back ["/* leading */", ""]. But then when evaluating
// the first argument of that binary expression (1) if you call this function you'll get ["/* leading */", ""]
// again. As such you should only call this for expressions where you know the sub expressions will not also
// pick up the same trivia. This is normally because of some token that makes up the current expression range.
// e.g. a ParenthesesExpr will have a range that includes the parentheses, but the sub expression will not. As
// such any trivia before and after those parens won't get picked up by any sub expression as they'll stop
// their trivia search at the parentheses.
//
// blockLike is used to tell the trivia search that this is searching for trivia for a block like expression.
// For example given a block of code like:
//
// locals {
//    # leading trivia
//    local_a = "a" /* trailing trivia a */
//
//    # leading trivia b
//    /* more leading trivia */ local_b = "b"
//    # trailing trivia
// }
//
// If we're searching for trivia for local_a and local_b we don't want to double count "trailing trivia a" and
// "leading trivia b". But there aren't any semantic tokens between these two blocks to split the trivia
// search. blockLike tells the search engine that it should treat new lines and braces as semantically
// significant and return ["# leading trivia", "/* trailing trivia a */"] for local_a, and ["# leading trivia
// b\n/* more leading trivia */", "# trailing trivia"] for local_b.

func getTrivia(sources map[string][]byte, r hcl.Range, blockLike bool) (hclwrite.Tokens, hclwrite.Tokens) {
	// Load the file referenced in the range
	src, has := sources[r.Filename]
	if !has {
		// This shouldn't ever be hit, "sources" is a list of every file we parsed earlier and ranges should
		// only come from those.
		panic(fmt.Sprintf("Could not read '%s' to parse trivia", r.Filename))
	}
	tokens, _ := hclsyntax.LexConfig(src, r.Filename, hcl.Pos{Byte: 0, Line: 1, Column: 1})
	// Ignore the diagnostics, we already know this is parsable because we've got the hcl.Range for it

	// Find the index of the first and last token matching the input
	var first, last int
	for i, token := range tokens {
		if token.Range.Start == r.Start {
			first = i
		}
		if token.Range.End == r.End {
			last = i
		}
	}

	return getTrivaFromIndex(tokens, first, last, blockLike)
}

// Given a HCL range for an attribute expression find the full range for that attribute
func getAttributeRange(sources map[string][]byte, r hcl.Range) hcl.Range {
	// Load the file referenced in the range
	src, has := sources[r.Filename]
	if !has {
		// This shouldn't ever be hit, "sources" is a list of every file we parsed earlier and ranges should
		// only come from those.
		panic(fmt.Sprintf("Could not read '%s' to parse trivia", r.Filename))
	}
	tokens, _ := hclsyntax.LexConfig(src, r.Filename, hcl.Pos{Byte: 0, Line: 1, Column: 1})
	// Ignore the diagnostics, we already know this is parsable because we've got the hcl.Range for it

	// Find the index of the first token matching the input range
	var first int
	for i, token := range tokens {
		if token.Range.Start == r.Start {
			first = i
			break
		}
	}

	// Now work backwards from first to find the identifier token for this attribute
	first = first - 1
	for first >= 0 {
		if tokens[first].Type == hclsyntax.TokenIdent {
			break
		}
		first = first - 1
	}

	return hcl.Range{
		Filename: r.Filename,
		Start:    tokens[first].Range.Start,
		End:      r.End,
	}
}

// Given a HCL range return the tokens for that range
func getTokensForRange(sources map[string][]byte, r hcl.Range) hclwrite.Tokens {
	// Load the file referenced in the range
	src, has := sources[r.Filename]
	if !has {
		// This shouldn't ever be hit, "sources" is a list of every file we parsed earlier and ranges should
		// only come from those.
		panic(fmt.Sprintf("Could not read '%s' to parse trivia", r.Filename))
	}
	tokens, _ := hclsyntax.LexConfig(src, r.Filename, hcl.Pos{Byte: 0, Line: 1, Column: 1})
	// Ignore the diagnostics, we already know this is parsable because we've got the hcl.Range for it

	// Find the tokens for this range
	rangeTokens := make(hclwrite.Tokens, 0)
	foundFirst := false
	for _, token := range tokens {
		if token.Range.Start == r.Start {
			foundFirst = true
		}

		if foundFirst {
			rangeTokens = append(rangeTokens, &hclwrite.Token{
				Type:  token.Type,
				Bytes: token.Bytes,
			})
		}

		if token.Range.End == r.End {
			break
		}
	}

	return rangeTokens
}

// tokens _must_ be tokens representing an expression that resolves to a list. This function will return a new
// set of tokens that represent a singleton value containing the first element of the input list.
func projectListToSingleton(tokens hclwrite.Tokens) hclwrite.Tokens {
	// See if this is a list literal expression, i.e. the first (non triva) token is an open bracket, and the
	// last is a close bracket.
	openBrack := -1
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		if isTrivia(token.Type) {
			continue
		}
		if token.Type == hclsyntax.TokenOBrack {
			openBrack = i
		}
		break
	}

	closeBrack := -1
	for i := len(tokens) - 1; i >= 0; i-- {
		token := tokens[i]
		if isTrivia(token.Type) {
			continue
		}
		if token.Type == hclsyntax.TokenCBrack {
			closeBrack = i
		}
		break
	}

	if openBrack == -1 || closeBrack == -1 {
		// Not a simple list literal, just return the input indexed at 0
		zeroIndex := hclwrite.Tokens{
			&hclwrite.Token{Type: hclsyntax.TokenOBrack, Bytes: []byte("[")},
			&hclwrite.Token{Type: hclsyntax.TokenNumberLit, Bytes: []byte("0")},
			&hclwrite.Token{Type: hclsyntax.TokenCBrack, Bytes: []byte("]")},
		}
		return append(tokens, zeroIndex...)
	}

	// We have a list literal, so we return a new list literal with just the element within it
	newTokens := make(hclwrite.Tokens, 0)
	for i, token := range tokens {
		if i == openBrack || i == closeBrack {
			continue
		}
		newTokens = append(newTokens, token)
	}
	return newTokens
}

// Functions need to translate in one of four ways
// 1. The `list` function just gets translated into a tuple
// 2. Just a simple rename, e.g. "file" => "readFile"
// 3. A translation to an std invoke
// 4. Left exactly as is (e.g. length), this is a no-op rename.
// 5. Assumed to be not supported.
var tfFunctionRenames = map[string]string{
	"sensitive":  "secret",
	"jsonencode": "toJSON",
	"length":     "length",
	"element":    "element",
}

var tfFunctionStd = map[string]struct {
	token  string
	inputs []string
	output string

	// True if and only if the function has a variable number of arguments. If this is the case, these arguments will be
	// packed into a single list in order to be passed to the relevant `pulumi-std` invoke, since invokes do not support
	// variadic arguments. If the function accepts more than one argument, the last argument is assumed to be the one that
	// requires packing.
	paramArgs bool
}{
	"abs": {
		token:  "std:index:abs",
		inputs: []string{"input"},
		output: "result",
	},
	"abspath": {
		token:  "std:index:abspath",
		inputs: []string{"input"},
		output: "result",
	},
	"alltrue": {
		token:  "std:index:alltrue",
		inputs: []string{"input"},
		output: "result",
	},
	"anytrue": {
		token:  "std:index:anytrue",
		inputs: []string{"input"},
		output: "result",
	},
	"base64decode": {
		token:  "std:index:base64decode",
		inputs: []string{"input"},
		output: "result",
	},
	"base64encode": {
		token:  "std:index:base64encode",
		inputs: []string{"input"},
		output: "result",
	},
	"base64gzip": {
		token:  "std:index:base64gzip",
		inputs: []string{"input"},
		output: "result",
	},
	"base64sha256": {
		token:  "std:index:base64sha256",
		inputs: []string{"input"},
		output: "result",
	},
	"base64sha512": {
		token:  "std:index:base64sha512",
		inputs: []string{"input"},
		output: "result",
	},
	"basename": {
		token:  "std:index:basename",
		inputs: []string{"input"},
		output: "result",
	},
	"bcrypt": {
		token:  "std:index:bcrypt",
		inputs: []string{"input", "cost"},
		output: "result",
	},
	"ceil": {
		token:  "std:index:ceil",
		inputs: []string{"input"},
		output: "result",
	},
	"chomp": {
		token:  "std:index:chomp",
		inputs: []string{"input"},
		output: "result",
	},
	"chunklist": {
		token:  "std:index:chunklist",
		inputs: []string{"input", "size"},
		output: "result",
	},
	"coalesce": {
		token:     "std:index:coalesce",
		inputs:    []string{"input"},
		output:    "result",
		paramArgs: true,
	},
	"coalescelist": {
		token:     "std:index:coalescelist",
		inputs:    []string{"input"},
		output:    "result",
		paramArgs: true,
	},
	"concat": {
		token:     "std:index:concat",
		inputs:    []string{"input"},
		output:    "result",
		paramArgs: true,
	},
	"contains": {
		token:  "std:index:contains",
		inputs: []string{"input", "element"},
		output: "result",
	},
	"compact": {
		token:  "std:index:compact",
		inputs: []string{"input"},
		output: "result",
	},
	"cidrhost": {
		token:  "std:index:cidrhost",
		inputs: []string{"input", "host"},
		output: "result",
	},
	"cidrnetmask": {
		token:  "std:index:cidrnetmask",
		inputs: []string{"input"},
		output: "result",
	},
	"cidrsubnet": {
		token:  "std:index:cidrsubnet",
		inputs: []string{"input", "newbits", "netnum"},
		output: "result",
	},
	"cidrsubnets": {
		token:     "std:index:cidrsubnets",
		inputs:    []string{"input", "newbits"},
		output:    "result",
		paramArgs: true,
	},
	"csvdecode": {
		token:  "std:index:csvdecode",
		inputs: []string{"input"},
		output: "result",
	},
	"dirname": {
		token:  "std:index:dirname",
		inputs: []string{"input"},
		output: "result",
	},
	"distinct": {
		token:  "std:index:distinct",
		inputs: []string{"input"},
		output: "result",
	},
	"endswith": {
		token:  "std:index:endswith",
		inputs: []string{"input", "suffix"},
		output: "result",
	},
	"file": {
		token:  "std:index:file",
		inputs: []string{"input"},
		output: "result",
	},
	"filebase64": {
		token:  "std:index:filebase64",
		inputs: []string{"input"},
		output: "result",
	},
	"filebase64sha256": {
		token:  "std:index:filebase64sha256",
		inputs: []string{"input"},
		output: "result",
	},
	"filebase64sha512": {
		token:  "std:index:filebase64sha512",
		inputs: []string{"input"},
		output: "result",
	},
	"fileexists": {
		token:  "std:index:fileexists",
		inputs: []string{"input"},
		output: "result",
	},
	"filemd5": {
		token:  "std:index:filemd5",
		inputs: []string{"input"},
		output: "result",
	},
	"filesha1": {
		token:  "std:index:filesha1",
		inputs: []string{"input"},
		output: "result",
	},
	"filesha256": {
		token:  "std:index:filesha256",
		inputs: []string{"input"},
		output: "result",
	},
	"filesha512": {
		token:  "std:index:filesha512",
		inputs: []string{"input"},
		output: "result",
	},
	"flatten": {
		token:  "std:index:flatten",
		inputs: []string{"input"},
		output: "result",
	},
	"floor": {
		token:  "std:index:floor",
		inputs: []string{"input"},
		output: "result",
	},
	"format": {
		token:     "std:index:format",
		inputs:    []string{"input", "args"},
		output:    "result",
		paramArgs: true,
	},
	"formatlist": {
		token:     "std:index:formatlist",
		inputs:    []string{"input", "args"},
		output:    "result",
		paramArgs: true,
	},
	"indent": {
		token:  "std:index:indent",
		inputs: []string{"spaces", "input"},
		output: "result",
	},
	"join": {
		token:  "std:index:join",
		inputs: []string{"separator", "input"},
		output: "result",
	},
	"jsondecode": {
		token:  "std:index:jsondecode",
		inputs: []string{"input"},
		output: "result",
	},
	"keys": {
		token:  "std:index:keys",
		inputs: []string{"input"},
		output: "result",
	},
	"log": {
		token:  "std:index:log",
		inputs: []string{"base", "input"},
		output: "result",
	},
	"lookup": {
		token:  "std:index:lookup",
		inputs: []string{"map", "key", "default"},
		output: "result",
	},
	"lower": {
		token:  "std:index:lower",
		inputs: []string{"input"},
		output: "result",
	},
	"max": {
		token:     "std:index:max",
		inputs:    []string{"input"},
		output:    "result",
		paramArgs: true,
	},
	"md5": {
		token:  "std:index:md5",
		inputs: []string{"input"},
		output: "result",
	},
	"merge": {
		token:     "std:index:merge",
		inputs:    []string{"input"},
		output:    "result",
		paramArgs: true,
	},
	"min": {
		token:     "std:index:min",
		inputs:    []string{"input"},
		output:    "result",
		paramArgs: true,
	},
	"parseint": {
		token:  "std:index:parseint",
		inputs: []string{"input", "base"},
		output: "result",
	},
	"pathexpand": {
		token:  "std:index:pathexpand",
		inputs: []string{"input"},
		output: "result",
	},
	"pow": {
		token:  "std:index:pow",
		inputs: []string{"base", "exponent"},
		output: "result",
	},
	"range": {
		token:  "std:index:range",
		inputs: []string{"limit", "start", "step"},
		output: "result",
	},
	"regex": {
		token:  "std:index:regex",
		inputs: []string{"pattern", "string"},
		output: "result",
	},
	"regexall": {
		token:  "std:index:regexall",
		inputs: []string{"pattern", "string"},
		output: "result",
	},
	"replace": {
		token:  "std:index:replace",
		inputs: []string{"text", "search", "replace"},
		output: "result",
	},
	"rsadecrypt": {
		token:  "std:index:rsadecrypt",
		inputs: []string{"cipherText", "key"},
		output: "result",
	},
	"setintersection": {
		token:     "std:index:setintersection",
		inputs:    []string{"input"},
		output:    "result",
		paramArgs: true,
	},
	"sha1": {
		token:  "std:index:sha1",
		inputs: []string{"input"},
		output: "result",
	},
	"sha256": {
		token:  "std:index:sha256",
		inputs: []string{"input"},
		output: "result",
	},
	"sha512": {
		token:  "std:index:sha512",
		inputs: []string{"input"},
		output: "result",
	},
	"signum": {
		token:  "std:index:signum",
		inputs: []string{"input"},
		output: "result",
	},
	"slice": {
		token:  "std:index:slice",
		inputs: []string{"list", "from", "to"},
		output: "result",
	},
	"sort": {
		token:  "std:index:sort",
		inputs: []string{"input"},
		output: "result",
	},
	"split": {
		token:  "std:index:split",
		inputs: []string{"separator", "text"},
		output: "result",
	},
	"startswith": {
		token:  "std:index:startswith",
		inputs: []string{"input", "prefix"},
		output: "result",
	},
	"strrev": {
		token:  "std:index:strrev",
		inputs: []string{"input"},
		output: "result",
	},
	"substr": {
		token:  "std:index:substr",
		inputs: []string{"input", "offset", "length"},
		output: "result",
	},
	"sum": {
		token:  "std:index:sum",
		inputs: []string{"input"},
		output: "result",
	},
	"timeadd": {
		token:  "std:index:timeadd",
		inputs: []string{"duration", "timestamp"},
		output: "result",
	},
	"timecmp": {
		token:  "std:index:timecmp",
		inputs: []string{"timestampa", "timestampb"},
		output: "result",
	},
	"timestamp": {
		token:  "std:index:timestamp",
		inputs: []string{},
		output: "result",
	},
	"title": {
		token:  "std:index:title",
		inputs: []string{"input"},
		output: "result",
	},
	"tobool": {
		token:  "std:index:tobool",
		inputs: []string{"input"},
		output: "result",
	},
	"toset": {
		token:  "std:index:toset",
		inputs: []string{"input"},
		output: "result",
	},
	"transpose": {
		token:  "std:index:transpose",
		inputs: []string{"input"},
		output: "result",
	},
	"trim": {
		token:  "std:index:trim",
		inputs: []string{"input", "cutset"},
		output: "result",
	},
	"trimprefix": {
		token:  "std:index:trimprefix",
		inputs: []string{"input", "prefix"},
		output: "result",
	},
	"trimspace": {
		token:  "std:index:trimspace",
		inputs: []string{"input"},
		output: "result",
	},
	"trimsuffix": {
		token:  "std:index:trimsuffix",
		inputs: []string{"input", "suffix"},
		output: "result",
	},
	"upper": {
		token:  "std:index:upper",
		inputs: []string{"input"},
		output: "result",
	},
	"urlencode": {
		token:  "std:index:urlencode",
		inputs: []string{"input"},
		output: "result",
	},
	"uuid": {
		token:  "std:index:uuid",
		inputs: []string{},
		output: "result",
	},
}

type convertState struct {
	// The sources for the HCL files we're converting
	sources map[string][]byte

	// Diagnostic messages from conversion
	diagnostics hcl.Diagnostics

	// Determines whether converting objects should rewrite keys to camelCase or keep it as is
	rewriteObjectKeys bool

	// Names that were changed to avoid overlaps with pcl keywords.
	renames     pclOverlapRenames
	typeRenames pclOverlapRenames

	sandboxedModuleNames map[string]string
}

type pclOverlapRenames struct {
	nameToRename map[string]string
	renameToName map[string]string
}

func (s *convertState) disableRewritingObjectKeys(f func()) {
	previous := s.rewriteObjectKeys
	s.rewriteObjectKeys = false
	f()
	s.rewriteObjectKeys = previous
}

// Adds a diagnostic to the state
func (s *convertState) appendDiagnostic(diagnostic *hcl.Diagnostic) {
	s.diagnostics = append(s.diagnostics, diagnostic)
}

// Returns the source code for the given range
func (s *convertState) sourceCode(rng hcl.Range) string {
	buffer := bytes.NewBufferString("")
	_, err := getTokensForRange(s.sources, rng).WriteTo(buffer)
	contract.AssertNoErrorf(err, "Failed to write tokens for range %v", rng)
	return strings.Replace(buffer.String(), "\r\n", "\n", -1)
}

func (s *convertState) renamePclOverlap(
	kind string,
	hclType *string,
	name string,
	hclRange *hcl.Range,
) (string, string) {
	newName := name
	newType := new(string)
	if hclType != nil {
		*newType = *hclType
	} else {
		*newType = ""
	}

	if hclType != nil && isPclKeyword(*hclType) {
		*newType = fmt.Sprintf("%s_%sType_", *hclType, kind)
		s.typeRenames.nameToRename[name] = *newType
		s.typeRenames.renameToName[newName] = name

		s.appendDiagnostic(&hcl.Diagnostic{
			Subject:  hclRange,
			Severity: hcl.DiagWarning,
			Summary:  kind + " type renamed to prevent keyword overlap",
			Detail:   fmt.Sprintf("Renaming %s type %s to %s to prevent overlap", kind, *hclType, *newType),
		})
	}

	if isPclKeyword(name) {
		newName = fmt.Sprintf("%s_%s_", name, kind)
		s.renames.nameToRename[name] = newName
		s.renames.renameToName[newName] = name

		s.appendDiagnostic(&hcl.Diagnostic{
			Subject:  hclRange,
			Severity: hcl.DiagWarning,
			Summary:  kind + " renamed to prevent keyword overlap",
			Detail:   fmt.Sprintf("Renaming %s %s of type %s to %s to prevent overlap", kind, name, *hclType, newName),
		})
	}

	return *newType, newName
}

func (s *convertState) getNewNameIfRenamed(name string) string {
	if newName, ok := s.renames.nameToRename[name]; ok {
		return newName
	}
	return name
}

func (s *convertState) getOriginalNameIfRenamed(name string) string {
	if newName, ok := s.renames.renameToName[name]; ok {
		return newName
	}
	return name
}

func isPclKeyword(s string) bool {
	switch s {
	case "for", "if", "else":
		return true
	default:
		return false
	}
}

// Returns a call to notImplemented with the text of the input range, e.g. `notImplemented("some.expr[0]")`
func notImplemented(state *convertState, rng hcl.Range) hclwrite.Tokens {
	text := cty.StringVal(state.sourceCode(rng))
	return hclwrite.TokensForFunctionCall("notImplemented", hclwrite.TokensForValue(text))
}

func convertFunctionCallExpr(state *convertState,
	scopes *scopes, fullyQualifiedPath string, call *hclsyntax.FunctionCallExpr,
) hclwrite.Tokens {
	callRange := hcl.RangeOver(call.NameRange, call.CloseParenRange)

	args := []hclwrite.Tokens{}
	for _, arg := range call.Args {
		if call.Name == "jsonencode" {
			// when encountering a jsonencode function, we need to convert the underlying content without
			// rewriting the object keys to camelCase (for example when converting JSON policy document of an IAM role)
			state.disableRewritingObjectKeys(func() {
				args = append(args, convertExpression(state, false, scopes, "", arg))
			})
		} else {
			args = append(args, convertExpression(state, false, scopes, "", arg))
		}
	}

	// First see if this is `list`
	if call.Name == "list" {
		listTokens := hclwrite.Tokens{makeToken(hclsyntax.TokenOBrack, "[")}
		first := true
		for _, arg := range args {
			if !first {
				listTokens = append(listTokens, makeToken(hclsyntax.TokenComma, ", "))
			}
			first = false
			listTokens = append(listTokens, arg...)
		}
		listTokens = append(listTokens, makeToken(hclsyntax.TokenCBrack, "]"))
		return listTokens
	}

	// Translate tolist(x) as x - in TF this normalizes sets to lists, but in Pulumi everything is represented as a
	// list anyway so a no-op is warranted.
	if call.Name == "tolist" && len(args) == 1 {
		return args[0]
	}

	// Next see if this is a rename
	if newName, has := tfFunctionRenames[call.Name]; has {
		return hclwrite.TokensForFunctionCall(newName, args...)
	}

	// Next see if it's mapped to a PCL invoke
	if invoke, has := tfFunctionStd[call.Name]; has {
		invokeArgs := make([]hclwrite.ObjectAttrTokens, 0)

		// If paramArgs has been set, we've been asked to pack all arguments after the first into a list. However, if the
		// call has the `ExpandFinal` attribute set to true, that means that it is of the form `f(x, y, zs...)` where `zs`
		// is an _already packed_ list of arguments that is to be spread into a variable number of arguments. Consequently,
		// in this case, we actually don't need to do anything, as using that last (unspread) value as-is will just work.
		// So, for example:
		//
		// f(1, 2, [3, 4]...) -> { x = 1, y = 2, zs = [3, 4] }
		//   The last argument was already packed; use it as-is
		//
		// f(1, 2, 3, 4)   -> { x = 1, y = 2, zs = [3, 4] }
		//   Last arguments were not packed; pack them into a list
		if invoke.paramArgs && !call.ExpandFinal {
			// We only want to pack values into the last formal argument. Everything up until then we can treat as positional.
			// So e.g. if the function's formal inputs are [a, b], and we are given [1, 2, 3, 4], we want to pass 1 for a and
			// pack up [2, 3, 4] into a list for b.
			i := 0
			for ; i < len(invoke.inputs)-1; i++ {
				invokeArgs = append(invokeArgs, hclwrite.ObjectAttrTokens{
					Name:  hclwrite.TokensForIdentifier(invoke.inputs[i]),
					Value: args[i],
				})
			}

			listTokens := hclwrite.Tokens{makeToken(hclsyntax.TokenOBrack, "[")}
			for j := i; j < len(args); j++ {
				if j > i {
					listTokens = append(listTokens, makeToken(hclsyntax.TokenComma, ","))
				}
				listTokens = append(listTokens, args[j]...)
			}
			listTokens = append(listTokens, makeToken(hclsyntax.TokenCBrack, "]"))

			invokeArgs = append(invokeArgs, hclwrite.ObjectAttrTokens{
				Name:  hclwrite.TokensForIdentifier(invoke.inputs[i]),
				Value: listTokens,
			})
		} else {
			if len(args) > len(invoke.inputs) {
				state.appendDiagnostic(&hcl.Diagnostic{
					Subject:  &callRange,
					Severity: hcl.DiagWarning,
					Summary:  "Unexpected argument count to function",
					Detail:   fmt.Sprintf("Got %d arguments to function %s, expected %d", len(args), call.Name, len(invoke.inputs)),
				})
			}

			for i, arg := range args {
				var name hclwrite.Tokens
				if i < len(invoke.inputs) {
					name = hclwrite.TokensForIdentifier(invoke.inputs[i])
				} else {
					name = hclwrite.TokensForIdentifier(fmt.Sprintf("arg%d", i))
				}

				invokeArgs = append(invokeArgs, hclwrite.ObjectAttrTokens{
					Name:  name,
					Value: arg,
				})
			}
		}
		token := hclwrite.TokensForValue(cty.StringVal(invoke.token))
		call := hclwrite.TokensForFunctionCall("invoke", token, hclwrite.TokensForObject(invokeArgs))

		if invoke.output == "" {
			return call
		}

		// Add an attribute access to get the output
		call = append(call, makeToken(hclsyntax.TokenDot, "."))
		call = append(call, makeToken(hclsyntax.TokenIdent, invoke.output))
		return call
	}

	// Finally just return it as not yet implemented
	state.appendDiagnostic(&hcl.Diagnostic{
		Subject:  &callRange,
		Severity: hcl.DiagWarning,
		Summary:  "Function not yet implemented",
		Detail:   fmt.Sprintf("Function %s not yet implemented, see %s", call.Name, getTrackingBug(call)),
	})

	return notImplemented(state, call.Range())
}

func convertTupleConsExpr(state *convertState, inBlock bool, scopes *scopes,
	fullyQualifiedPath string, expr *hclsyntax.TupleConsExpr,
) hclwrite.Tokens {
	elems := []hclwrite.Tokens{}
	for _, expr := range expr.Exprs {
		elems = append(elems, convertExpression(state, false, scopes, appendPathArray(fullyQualifiedPath), expr))
	}
	tokens := hclwrite.TokensForTuple(elems)
	leading, trailing := getTrivia(state.sources, expr.SrcRange, inBlock)
	return append(append(leading, tokens...), trailing...)
}

// appendPath appends a part to a fully quailifed dot-separated path. If the root is "" then append returns
// "". If the part is "" then append returns root.
func appendPath(root, part string) string {
	if root == "" || part == "" {
		return ""
	}
	return root + "." + part
}

// appendPathArray appends an array part to a fully quailifed dot-separated path. If the root is "" then append returns
// "".
func appendPathArray(root string) string {
	if root == "" {
		return ""
	}
	return root + "[]"
}

// matchStaticString returns a literal string if the expression is a static string or identifier, else nil. It returns
// true if this was an identifier.
func matchStaticString(expr hclsyntax.Expression) (*string, bool) {
	switch expr := expr.(type) {
	case *hclsyntax.ObjectConsKeyExpr:
		return matchStaticString(expr.Wrapped)
	case *hclsyntax.LiteralValueExpr:
		if expr.Val.Type() != cty.String {
			return nil, false
		}
		s := expr.Val.AsString()
		return &s, false
	case *hclsyntax.ScopeTraversalExpr:
		if len(expr.Traversal) != 1 {
			return nil, false
		}
		if root, ok := expr.Traversal[0].(hcl.TraverseRoot); ok {
			s := root.Name
			return &s, true
		}
	case *hclsyntax.TemplateExpr:
		if len(expr.Parts) != 1 {
			return nil, false
		}
		if lit, ok := expr.Parts[0].(*hclsyntax.LiteralValueExpr); ok {
			if lit.Val.Type() != cty.String {
				return nil, false
			}
			s := lit.Val.AsString()
			return &s, false
		}
	}
	return nil, false
}

func convertObjectConsExpr(state *convertState, inBlock bool, scopes *scopes,
	fullyQualifiedPath string, expr *hclsyntax.ObjectConsExpr,
) hclwrite.Tokens {
	items := []hclwrite.ObjectAttrTokens{}

	// If the object has at least one key that is not an identifier, it must be a map.
	hasNonIdentifierKey := false
	for _, item := range expr.Items {
		if _, isIdentifier := matchStaticString(item.KeyExpr); !isIdentifier {
			hasNonIdentifierKey = true
			break
		}
	}

	for _, item := range expr.Items {
		// Keys _might_ need renaming if we're translating for an object type, we can do this if it's
		// statically known and we know our current path
		var nameTokens hclwrite.Tokens
		var subQualifiedPath string
		name, isIdentifier := matchStaticString(item.KeyExpr)
		if fullyQualifiedPath != "" {
			// We should rename the object keys if this is an object type. It's a map type we should leave it
			// alone.
			isMap := scopes.isMap(fullyQualifiedPath)
			if name != nil {
				if isMap != nil && *isMap || hasNonIdentifierKey {
					// We know this _is_ a map type, so we should leave the keys alone. We can only do this for static
					// strings other expressions are just going to have to be converted.
				} else {
					// We either don't know this type, or know it's not a map, so we should try to rename the keys.
					subQualifiedPath = appendPath(fullyQualifiedPath, *name)
					if state.rewriteObjectKeys {
						*name = scopes.pulumiName(*name, subQualifiedPath)
					}
				}

				// If this was a literal string (i.e. not an identifier) then make sure we still quote it.
				if !isIdentifier {
					nameTokens = hclwrite.TokensForValue(cty.StringVal(*name))
				} else {
					nameTokens = hclwrite.TokensForIdentifier(*name)
				}
			}
		}
		// If we can't statically determine the name, we can't rename it, so just convert the expression.
		if nameTokens == nil {
			if name != nil && !state.rewriteObjectKeys {
				nameTokens = hclwrite.TokensForValue(cty.StringVal(*name))
			} else {
				nameTokens = convertExpression(state, false, scopes, "", item.KeyExpr)
			}
		}

		valueTokens := convertExpression(state, false, scopes, subQualifiedPath, item.ValueExpr)
		items = append(items, hclwrite.ObjectAttrTokens{
			Name:  nameTokens,
			Value: valueTokens,
		})
	}
	return hclwrite.TokensForObject(items)
}

func convertObjectConsKeyExpr(state *convertState, inBlock bool,
	scopes *scopes, fullyQualifiedPath string, expr *hclsyntax.ObjectConsKeyExpr,
) hclwrite.Tokens {
	// Seems we can just ignore ForceNonLiteral here
	return convertExpression(state, false, scopes, fullyQualifiedPath, expr.Wrapped)
}

func convertLiteralValueExpr(state *convertState, inBlock bool,
	expr *hclsyntax.LiteralValueExpr,
) hclwrite.Tokens {
	leading, trailing := getTrivia(state.sources, expr.SrcRange, false)
	value := hclwrite.TokensForValue(expr.Val)

	tokens := leading
	tokens = append(tokens, value...)
	tokens = append(tokens, trailing...)
	return tokens
}

func makeToken(typ hclsyntax.TokenType, str string) *hclwrite.Token {
	return &hclwrite.Token{
		Type:  typ,
		Bytes: []byte(str),
	}
}

func convertTemplateWrapExpr(state *convertState,
	scopes *scopes, fullyQualifiedPath string, expr *hclsyntax.TemplateWrapExpr,
) hclwrite.Tokens {
	tokens := []*hclwrite.Token{}
	tokens = append(tokens, makeToken(hclsyntax.TokenOQuote, "\""))
	tokens = append(tokens, makeToken(hclsyntax.TokenTemplateInterp, "${"))
	tokens = append(tokens, convertExpression(state, false, scopes, "", expr.Wrapped)...)
	tokens = append(tokens, makeToken(hclsyntax.TokenTemplateSeqEnd, "}"))
	tokens = append(tokens, makeToken(hclsyntax.TokenCQuote, "\""))
	return tokens
}

// convertTemplateJoinExpr writes "template joins" back out as they are.  These
// are for expressions in the HCL template syntax like so:
// "%{for foo in local.bars~}foo%{endfor~}
func convertTemplateJoinExpr(state *convertState, inBlock bool, scopes *scopes,
	fullyQualifiedPath string, expr *hclsyntax.TemplateJoinExpr,
) hclwrite.Tokens {
	forExpr, ok := expr.Tuple.(*hclsyntax.ForExpr)
	if !ok {
		contract.Failf("Template Join Expression must contain for expression, but contains %T", expr.Tuple)
	}

	// The collection doesn't yet have access to the key/value scopes
	collTokens := convertExpression(state, false, scopes, "", forExpr.CollExpr)

	locals := map[string]string{
		forExpr.ValVar: camelCaseName(forExpr.ValVar),
	}
	if forExpr.KeyVar != "" {
		locals[forExpr.KeyVar] = camelCaseName(forExpr.KeyVar)
	}
	scopes.push(locals)

	valueTokens := convertExpression(state, false, scopes, "", forExpr.ValExpr)
	valueTokens = valueTokens[1 : len(valueTokens)-1] // strip the outer quote tokens

	scopes.pop()

	var tokens hclwrite.Tokens

	// Write the opening of the for template
	tokens = append(tokens, makeToken(hclsyntax.TokenTemplateControl, "%{"))
	tokens = append(tokens, makeToken(hclsyntax.TokenIdent, "for"))
	if locals[forExpr.KeyVar] != "" {
		tokens = append(tokens, makeToken(hclsyntax.TokenIdent, locals[forExpr.KeyVar]))
		tokens = append(tokens, makeToken(hclsyntax.TokenComma, ","))
	}
	tokens = append(tokens, makeToken(hclsyntax.TokenIdent, locals[forExpr.ValVar]))
	tokens = append(tokens, makeToken(hclsyntax.TokenIdent, "in"))
	tokens = append(tokens, collTokens...)
	tokens = append(tokens, makeToken(hclsyntax.TokenTemplateControl, "~}"))

	// Write the inside of the for template
	tokens = append(tokens, valueTokens...)

	// Write the %{endfor~}
	tokens = append(tokens, makeToken(hclsyntax.TokenTemplateControl, "%{"))
	tokens = append(tokens, makeToken(hclsyntax.TokenIdent, "endfor"))
	tokens = append(tokens, makeToken(hclsyntax.TokenTemplateControl, "~}"))

	return tokens
}

func convertTemplateExpr(state *convertState,
	scopes *scopes, fullyQualifiedPath string, expr *hclsyntax.TemplateExpr,
) hclwrite.Tokens {
	tokens := []*hclwrite.Token{}
	oDelim, cDelim, isHereDoc := detectHeredocDelim(state, expr.Range())
	if isHereDoc {
		tokens = append(tokens, makeToken(hclsyntax.TokenOHeredoc, oDelim))
	} else {
		tokens = append(tokens, makeToken(hclsyntax.TokenOQuote, "\""))
	}
	for _, part := range expr.Parts {
		// If it's a literal then we can just write it to the string directly, else we need to wrap it in a
		// ${} block, or template controls.
		if lit, ok := part.(*hclsyntax.LiteralValueExpr); ok {
			if lit.Val.Type() == cty.String {
				// Strings get written directly without their surrounding quotes
				strtoks := hclwrite.TokensForValue(lit.Val)
				// strip the first and last token (")
				if len(strtoks) > 0 {
					strtoks = strtoks[1 : len(strtoks)-1]
				}

				var strtoksWithNewlines hclwrite.Tokens
				// Newlines should remain newlines in heredocs
				for _, tok := range strtoks {
					if tok.Type == hclsyntax.TokenQuotedLit && isHereDoc {
						splitStrings := strings.Split(string(tok.Bytes), "\\n")
						for i, splitString := range splitStrings {
							if splitString != "" {
								strtoksWithNewlines = append(strtoksWithNewlines, makeToken(hclsyntax.TokenStringLit, splitString))
							}
							if i < len(splitStrings)-1 {
								// Insert only true newlines.  This is when not the last newline
								// of the expression unless it is the end of the expression
								// (which must end in a newline)
								strtoksWithNewlines = append(strtoksWithNewlines, makeToken(hclsyntax.TokenStringLit, "\n"))
							}
						}
					} else {
						strtoksWithNewlines = append(strtoksWithNewlines, tok)
					}
				}

				tokens = append(tokens, strtoksWithNewlines...)
			} else {
				// Other values can be written as is
				tokens = append(tokens, hclwrite.TokensForValue(lit.Val)...)
			}
		} else if forToken, ok := part.(*hclsyntax.TemplateJoinExpr); ok {
			// A template join is more complex and contains template control
			// sections, so delegate to the inner expression.
			tokens = append(tokens, convertTemplateJoinExpr(state, false, scopes, "", forToken)...)
			if isHereDoc {
				tokens = append(tokens, makeToken(hclsyntax.TokenStringLit, "\n"))
			}
		} else {
			tokens = append(tokens, makeToken(hclsyntax.TokenTemplateInterp, "${"))
			tokens = append(tokens, convertExpression(state, false, scopes, "", part)...)
			tokens = append(tokens, makeToken(hclsyntax.TokenTemplateSeqEnd, "}"))
		}
	}
	if isHereDoc {
		tokens = append(tokens, makeToken(hclsyntax.TokenCHeredoc, cDelim+"\n"))
	} else {
		tokens = append(tokens, makeToken(hclsyntax.TokenCQuote, "\""))
	}
	return tokens
}

// detectHeredocDelim detects if the range contains a heredoc token and returns
// the open and close delimiters.
func detectHeredocDelim(state *convertState, r hcl.Range) (string, string, bool) {
	hereDocRegex := regexp.MustCompile(`^<<-?([[:alpha:]]+)\n`)

	file := state.sources[r.Filename]
	start := r.Start.Byte
	hereDocStr := hereDocRegex.FindSubmatch(file[start:])
	if hereDocStr != nil {
		// Return token without newline
		o := string(hereDocStr[0])
		clo := string(hereDocStr[1])
		return o, clo, true
	}

	return "", "", false
}

func camelCaseName(name string) string {
	// If name is all uppercase assume it stays upper case, else camel case it in pulumi style
	if strings.ToUpper(name) == name {
		return name
	}

	name = tfbridge.TerraformToPulumiNameV2(name, nil, nil)
	name = strings.ToLower(string(rune(name[0]))) + name[1:]
	return name
}

func pascalCaseName(name string) string {
	if name == "" {
		return ""
	}

	ccName := camelCaseName(name)
	return strings.ToUpper(string(rune(ccName[0]))) + ccName[1:]
}

// Returns whether the fully qualified path is being applied for a property.
func (s *scopes) isPropertyPath(fullyQualifiedPath string) bool {
	if fullyQualifiedPath == "" {
		return false
	}
	info, _ := s.getInfo(fullyQualifiedPath)
	return info.Resource == nil && info.ResourceInfo == nil && info.DataSourceInfo == nil
}

func rewriteRelativeTraversal(state *convertState,
	scopes *scopes, fullyQualifiedPath string, traversal hcl.Traversal,
) hcl.Traversal {
	if len(traversal) == 0 {
		return traversal
	}

	newTraversal := make([]hcl.Traverser, 0)
	if attr, ok := traversal[0].(hcl.TraverseAttr); ok {
		// An attribute look up, we need to know the type path of the traversal so far to resolve this correctly
		var name string
		if fullyQualifiedPath != "" {
			fullyQualifiedPath = appendPath(fullyQualifiedPath, attr.Name)
			name = scopes.pulumiName(attr.Name, fullyQualifiedPath)
		} else {
			name = tfbridge.TerraformToPulumiNameV2(attr.Name, nil, nil)
		}

		newTraversal = append(newTraversal, hcl.TraverseAttr{Name: name})
		newTraversal = append(newTraversal, rewriteRelativeTraversal(state, scopes, fullyQualifiedPath, traversal[1:])...)
	} else if index, ok := traversal[0].(hcl.TraverseIndex); ok {
		if scopes.isPropertyPath(fullyQualifiedPath) && scopes.maxItemsOne(fullyQualifiedPath) {
			// if are indexing a field which is marked with max items = 1
			// then we skip the index altogether and return traversal as is
			newTraversal = append(newTraversal, rewriteRelativeTraversal(state, scopes, fullyQualifiedPath, traversal[1:])...)
		} else {
			var indexOrAttrTraversal hcl.Traverser
			indexOrAttrTraversal = hcl.TraverseIndex{Key: index.Key}
			// If key is a static string and the container is an object (not a map),
			// prefer attribute access: myObject.someKey
			if fullyQualifiedPath != "" && scopes.isPropertyPath(fullyQualifiedPath) {
				if isMap := scopes.isMap(fullyQualifiedPath); !(isMap != nil && *isMap) {
					if state.rewriteObjectKeys {
						name := index.Key.AsString()
						fullyQualifiedPath = appendPath(fullyQualifiedPath, name)
						name = scopes.pulumiName(name, fullyQualifiedPath)
						indexOrAttrTraversal = hcl.TraverseAttr{Name: name}
					}
				}
			}
			// Index just translates as is
			newTraversal = append(newTraversal, indexOrAttrTraversal)
			newTraversal = append(newTraversal, rewriteRelativeTraversal(state, scopes, fullyQualifiedPath, traversal[1:])...)
		}
	} else {
		panic(fmt.Sprintf("Relative traverser %T not handled", traversal[0]))
	}

	return newTraversal
}

func getTraversalRange(traversal hcl.Traversal) hcl.Range {
	contract.Requiref(len(traversal) > 0, "traversal", "Traversal must have at least one element")

	rng := traversal[0].SourceRange()
	for _, t := range traversal[1:] {
		rng = hcl.RangeOver(rng, t.SourceRange())
	}
	return rng
}

func rewriteTraversal(
	state *convertState,
	scopes *scopes, fullyQualifiedPath string, traversal hcl.Traversal,
) hclwrite.Tokens {
	// We need to rewrite traversals, because we don't have the same top level variable names as terraform.
	contract.Requiref(len(traversal) > 0, "traversal", "Traversal must have at least one element")

	newTraversal := make([]hcl.Traverser, 0)

	var maybeFirstAttr *hcl.TraverseAttr
	if len(traversal) > 1 {
		if attr, ok := traversal[1].(hcl.TraverseAttr); ok {
			maybeFirstAttr = &attr
		}
	}
	var maybeSecondAttr *hcl.TraverseAttr
	if len(traversal) > 2 {
		if attr, ok := traversal[2].(hcl.TraverseAttr); ok {
			maybeSecondAttr = &attr
		}
	}

	contextRange := traversal.SourceRange()
	if root, ok := traversal[0].(hcl.TraverseRoot); ok {
		subjectRange := contextRange
		// If we hit an error it will be with the first attribute, not the root.
		if maybeFirstAttr != nil {
			subjectRange = maybeFirstAttr.SourceRange()
		}

		matches := func(rootName, attrName string) bool {
			return root.Name == rootName && maybeFirstAttr != nil && maybeFirstAttr.Name == attrName
		}

		if matches("terraform", "workspace") || matches("terraform", "applying") ||
			matches("path", "module") {
			// If this is one of the builtin terraform inputs we just rewrite it to notImplemented.
			state.appendDiagnostic(&hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  "Terraform input not yet implemented",
				Detail:   root.Name,
				Context:  &subjectRange,
				Subject:  &contextRange,
			})
			return notImplemented(state, getTraversalRange(traversal))
		} else if matches("path", "root") {
			return hclwrite.TokensForFunctionCall("rootDirectory")
		} else if matches("path", "cwd") {
			state.diagnostics = append(state.diagnostics, &hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  "Converting the builtin variable path.cwd with differing behavior",
				Detail: `The builtin Terraform variable path.cwd is being converted, but in Pulumi cwd will start at the ` +
					"\n" + `project program directory, not the execution directory`,
				Subject: &subjectRange,
				Context: &contextRange,
			})
			return hclwrite.TokensForFunctionCall("cwd")
		} else if root.Name == "var" && maybeFirstAttr != nil {
			// This is a lookup of a var etc, we need to rewrite this traversal such that the root is now the
			// pulumi config value instead.
			newName := scopes.getOrAddPulumiName("var."+maybeFirstAttr.Name, "", "Config")
			newTraversal = append(newTraversal, hcl.TraverseRoot{Name: newName})
			newTraversal = append(newTraversal, rewriteRelativeTraversal(state, scopes, "", traversal[2:])...)
		} else if root.Name == "local" && maybeFirstAttr != nil {
			// This is a lookup of a local etc, we need to rewrite this traversal such that the root is now the
			// pulumi local value instead.
			newName := scopes.getOrAddPulumiName("local."+maybeFirstAttr.Name, "their", "")
			newTraversal = append(newTraversal, hcl.TraverseRoot{Name: newName})
			newTraversal = append(newTraversal, rewriteRelativeTraversal(state, scopes, "", traversal[2:])...)
		} else if root.Name == "data" && maybeFirstAttr != nil && maybeSecondAttr != nil {
			// This is a lookup of a data resources etc, we need to rewrite this traversal such that the root is now the
			// pulumi invoked value instead.
			path := "data." + maybeFirstAttr.Name + "." + maybeSecondAttr.Name
			rootName := scopes.lookup(path)
			if rootName != "" {
				newName := scopes.getOrAddPulumiName(path, "", "data"+camelCaseName(rootName))
				newTraversal = append(newTraversal, hcl.TraverseRoot{Name: newName})
				newTraversal = append(newTraversal, rewriteRelativeTraversal(state, scopes, path, traversal[3:])...)
			} else {
				// unbound data source / invoke usage.
				// turn data.{data_source_token}.{local_name}.{rest}
				// into {localName}{DataSourceToken}.{rest}
				suffix := camelCaseName(maybeFirstAttr.Name)
				newRootName := scopes.getOrAddPulumiName(path, "", suffix)
				newTraversal = append(newTraversal, hcl.TraverseRoot{Name: newRootName})
				newTraversal = append(newTraversal, rewriteRelativeTraversal(state, scopes, "", traversal[3:])...)
			}
		} else if root.Name == "count" && maybeFirstAttr != nil {
			if maybeFirstAttr.Name == "index" && scopes.countIndex != nil {
				newTraversal = append(newTraversal, scopes.countIndex...)
				newTraversal = append(newTraversal, rewriteRelativeTraversal(state, scopes, "", traversal[2:])...)
			} else if maybeFirstAttr.Name != "index" {
				// Saw some other attribute on count, this is an error
				state.appendDiagnostic(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  `Invalid "count" attribute`,
					Detail: fmt.Sprintf(
						`The "count" object does not have an attribute named %q. `+
							`The only supported attribute is count.index, `+
							`which is the index of each instance of a resource block that has the "count" argument set.`,
						maybeFirstAttr.Name),
					Context: &subjectRange,
					Subject: &contextRange,
				})
				return nil
			} else {
				// We didn't have a count_index set but saw count.index!
				state.appendDiagnostic(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  `Reference to "count" in non-counted context`,
					Detail: `The "count" object can only be used in "module", "resource", and "data" blocks, ` +
						`and only when the "count" argument is set.`,
					Context: &subjectRange,
					Subject: &contextRange,
				})
				return nil
			}
		} else if root.Name == "each" && maybeFirstAttr != nil {
			// This _might_ be the special "each" value or it might just be a local, check the latter first
			localName := scopes.lookup("each")
			if localName != "" {
				newTraversal = append(newTraversal, hcl.TraverseRoot{Name: localName})
				newTraversal = append(newTraversal, rewriteRelativeTraversal(state, scopes, "", traversal[1:])...)
			} else {
				if maybeFirstAttr.Name == "key" {
					if scopes.eachKey != nil {
						newTraversal = append(newTraversal, scopes.eachKey...)
						newTraversal = append(newTraversal, rewriteRelativeTraversal(state, scopes, "", traversal[2:])...)
					} else {
						state.appendDiagnostic(&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  `Reference to "each" in context without for_each`,
							Detail: `The "each" object can be used only in "module" or "resource" blocks, ` +
								`and only when the "for_each" argument is set.`,
							Context: &subjectRange,
							Subject: &contextRange,
						})
						return nil
					}
				} else if maybeFirstAttr.Name == "value" {
					if scopes.eachValue != nil {
						newTraversal = append(newTraversal, scopes.eachValue...)
						newTraversal = append(newTraversal, rewriteRelativeTraversal(state, scopes, fullyQualifiedPath, traversal[2:])...)
					} else {
						state.appendDiagnostic(&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  `each.value cannot be used in this context`,
							Detail: `A reference to "each.value" has been used in a context in which it is unavailable, ` +
								`such as when the configuration no longer contains the value in its "for_each" expression. ` +
								`Remove this reference to each.value in your configuration to work around this error.`,
							Context: &subjectRange,
							Subject: &contextRange,
						})
						return nil
					}
				} else {
					state.appendDiagnostic(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  `Invalid "each" attribute`,
						Detail: fmt.Sprintf(
							`The "each" object does not have an attribute named %q. `+
								`The supported attributes are each.key and each.value, `+
								`the current key and value pair of the "for_each" attribute set.`, maybeFirstAttr.Name),
						Context: &subjectRange,
						Subject: &contextRange,
					})
					return nil
				}
			}
		} else if maybeFirstAttr != nil {
			// This is a lookup of a resource or an attribute lookup on a local variable etc, we need to
			// rewrite this traversal such that the root is now the pulumi invoked value instead.
			rewriteName := state.getNewNameIfRenamed(maybeFirstAttr.Name)

			// First see if this is a resource
			path := root.Name + "." + rewriteName
			newName := scopes.lookup(path)
			if newName != "" {
				// Looks like this is a resource because a local variable would not be recorded in scopes with a "." in it.
				newTraversal = append(newTraversal, hcl.TraverseRoot{Name: newName})
				newTraversal = append(newTraversal, rewriteRelativeTraversal(state, scopes, path, traversal[2:])...)
			} else {
				// This is either a local variable or a resource we haven't seen yet. First check if this is a local variable
				newName := scopes.lookup(root.Name)
				if newName != "" {
					// Looks like this is a local variable, just rewrite the rest of the traversal
					newTraversal = append(newTraversal, hcl.TraverseRoot{Name: newName})
					newTraversal = append(newTraversal, rewriteRelativeTraversal(state, scopes, "", traversal[1:])...)
				} else {
					// We don't know what this is, so let's assume it's an unknown resource (we shouldn't ever have unknown locals)
					// turn {resource_type}.{resource_name}.{rest}
					// into {resourceName}{ResourceType}.{rest}
					suffix := camelCaseName(root.Name)
					newRootName := scopes.getOrAddPulumiName(path, "", suffix)
					newTraversal = append(newTraversal, hcl.TraverseRoot{Name: newRootName})
					newTraversal = append(newTraversal, rewriteRelativeTraversal(state, scopes, "", traversal[2:])...)
				}
			}
		} else {
			// This is a lookup of a variable, look it up and use it else just us the name given
			newName := scopes.lookup(root.Name)
			if newName != "" {
				newTraversal = append(newTraversal, hcl.TraverseRoot{Name: newName})
				newTraversal = append(newTraversal, rewriteRelativeTraversal(state, scopes, "", traversal[1:])...)
			} else {
				// This will be an object key or an undeclared variable, try our best to rename those to match
				// pulumi style (i.e. camelCase)
				newTraversal = append(newTraversal, hcl.TraverseRoot{Name: camelCaseName(root.Name)})
				newTraversal = append(newTraversal, rewriteRelativeTraversal(state, scopes, "", traversal[1:])...)
			}
		}
	} else {
		panic(fmt.Sprintf("Root traverser %T not handled", traversal[0]))
	}

	return hclwrite.TokensForTraversal(newTraversal)
}

func isModuleRef(expr *hclsyntax.ScopeTraversalExpr) (string, bool) {
	root := expr.Traversal.RootName()
	if root != "module" {
		return "", false
	}
	if attr, ok := expr.Traversal[1].(hcl.TraverseAttr); ok {
		return attr.Name, true
	}
	return "", false
}

func convertScopeTraversalExpr(
	state *convertState, inBlock bool,
	scopes *scopes, fullyQualifiedPath string, expr *hclsyntax.ScopeTraversalExpr,
) hclwrite.Tokens {
	if moduleRef, ok := isModuleRef(expr); ok {
		if name, ok := state.sandboxedModuleNames[moduleRef]; ok {
			// sandboxed modules should not rewrite its traversal parts
			// because the outputs maintain their casing
			// so we only rewrite the module name
			// i.e. module.my_vpc.something_else (tf) becomes myVpc.something_else (PCL)
			var traversal hcl.Traversal
			newRoot := hcl.TraverseRoot{Name: name}
			traversal = append(traversal, newRoot)
			traversal = append(traversal, expr.Traversal[2:]...)
			return hclwrite.TokensForTraversal(traversal)
		}
	}
	return rewriteTraversal(state, scopes, fullyQualifiedPath, expr.Traversal)
}

func convertRelativeTraversalExpr(
	state *convertState, inBlock bool, scopes *scopes,
	fullyQualifiedPath string, expr *hclsyntax.RelativeTraversalExpr,
) hclwrite.Tokens {
	tokens := convertExpression(state, false, scopes, "", expr.Source)
	tokens = append(tokens, hclwrite.TokensForTraversal(
		rewriteRelativeTraversal(state, scopes, fullyQualifiedPath, expr.Traversal))...)
	return tokens
}

func convertBinaryOpExpr(state *convertState, inBlock bool, scopes *scopes,
	fullyQualifiedPath string, expr *hclsyntax.BinaryOpExpr,
) hclwrite.Tokens {
	tokens := convertExpression(state, false, scopes, fullyQualifiedPath, expr.LHS)
	switch expr.Op {
	case hclsyntax.OpLogicalOr:
		tokens = append(tokens, makeToken(hclsyntax.TokenOr, "||"))
	case hclsyntax.OpLogicalAnd:
		tokens = append(tokens, makeToken(hclsyntax.TokenAnd, "&&"))
	case hclsyntax.OpEqual:
		tokens = append(tokens, makeToken(hclsyntax.TokenEqualOp, "=="))
	case hclsyntax.OpNotEqual:
		tokens = append(tokens, makeToken(hclsyntax.TokenNotEqual, "!="))
	case hclsyntax.OpGreaterThan:
		tokens = append(tokens, makeToken(hclsyntax.TokenGreaterThan, ">"))
	case hclsyntax.OpGreaterThanOrEqual:
		tokens = append(tokens, makeToken(hclsyntax.TokenGreaterThanEq, ">="))
	case hclsyntax.OpLessThan:
		tokens = append(tokens, makeToken(hclsyntax.TokenLessThan, "<"))
	case hclsyntax.OpLessThanOrEqual:
		tokens = append(tokens, makeToken(hclsyntax.TokenLessThanEq, "<="))
	case hclsyntax.OpAdd:
		tokens = append(tokens, makeToken(hclsyntax.TokenPlus, "+"))
	case hclsyntax.OpSubtract:
		tokens = append(tokens, makeToken(hclsyntax.TokenMinus, "-"))
	case hclsyntax.OpMultiply:
		tokens = append(tokens, makeToken(hclsyntax.TokenStar, "*"))
	case hclsyntax.OpDivide:
		tokens = append(tokens, makeToken(hclsyntax.TokenSlash, "/"))
	case hclsyntax.OpModulo:
		tokens = append(tokens, makeToken(hclsyntax.TokenPercent, "%"))
	default:
		contract.Failf("unknown binary operation: %T", expr)
	}
	tokens = append(tokens, convertExpression(state, false, scopes, fullyQualifiedPath, expr.RHS)...)
	return tokens
}

func convertUnaryOpExpr(state *convertState, inBlock bool, scopes *scopes,
	fullyQualifiedPath string, expr *hclsyntax.UnaryOpExpr,
) hclwrite.Tokens {
	var tokens hclwrite.Tokens
	switch expr.Op {
	case hclsyntax.OpLogicalNot:
		tokens = hclwrite.Tokens{makeToken(hclsyntax.TokenBang, "!")}
	case hclsyntax.OpNegate:
		tokens = hclwrite.Tokens{makeToken(hclsyntax.TokenMinus, "-")}
	default:
		contract.Failf("unknown unary operation: %T", expr)
	}
	tokens = append(tokens, convertExpression(state, false, scopes, fullyQualifiedPath, expr.Val)...)
	return tokens
}

func convertForExpr(state *convertState, inBlock bool, scopes *scopes,
	fullyQualifiedPath string, expr *hclsyntax.ForExpr,
) hclwrite.Tokens {
	// The collection doesn't yet have access to the key/value scopes
	collTokens := convertExpression(state, false, scopes, "", expr.CollExpr)

	// TODO: We should ensure key and value vars are unique
	locals := map[string]string{
		expr.ValVar: camelCaseName(expr.ValVar),
	}
	if expr.KeyVar != "" {
		locals[expr.KeyVar] = camelCaseName(expr.KeyVar)
	}
	scopes.push(locals)

	keyTokens := convertExpression(state, false, scopes, "", expr.KeyExpr)
	valueTokens := convertExpression(state, false, scopes, "", expr.ValExpr)
	condTokens := convertExpression(state, false, scopes, "", expr.CondExpr)

	scopes.pop()

	// Translate to either a tuple or object expression
	// ForExpr = forTupleExpr | forObjectExpr;
	// forTupleExpr = "[" forIntro Expression forCond? "]";
	// forObjectExpr = "{" forIntro Expression "=>" Expression "..."? forCond? "}";
	// forIntro = "for" Identifier ("," Identifier)? "in" Expression ":";
	// forCond = "if" Expression;
	var tokens hclwrite.Tokens
	if keyTokens == nil {
		// This is a forTupleExpr
		tokens = hclwrite.Tokens{makeToken(hclsyntax.TokenOBrack, "[")}
	} else {
		// This is a forObjectExpr
		tokens = hclwrite.Tokens{makeToken(hclsyntax.TokenOBrace, "{")}
	}

	// Write the intro
	tokens = append(tokens, makeToken(hclsyntax.TokenIdent, "for"))
	if locals[expr.KeyVar] != "" {
		tokens = append(tokens, makeToken(hclsyntax.TokenIdent, locals[expr.KeyVar]))
		tokens = append(tokens, makeToken(hclsyntax.TokenComma, ","))
	}
	tokens = append(tokens, makeToken(hclsyntax.TokenIdent, locals[expr.ValVar]))
	tokens = append(tokens, makeToken(hclsyntax.TokenIdent, "in"))
	tokens = append(tokens, collTokens...)
	tokens = append(tokens, makeToken(hclsyntax.TokenColon, ":"))

	// Write the key part (if present)
	if keyTokens != nil {
		tokens = append(tokens, keyTokens...)
		tokens = append(tokens, makeToken(hclsyntax.TokenFatArrow, "=>"))
	}

	// Write the value part
	tokens = append(tokens, valueTokens...)

	// Write the ellipsis part (if present)
	if expr.Group {
		tokens = append(tokens, makeToken(hclsyntax.TokenEllipsis, "..."))
	}

	// Write the conditional part (if present)
	if condTokens != nil {
		tokens = append(tokens, makeToken(hclsyntax.TokenIdent, "if"))
		tokens = append(tokens, condTokens...)
	}

	// Write the end
	if keyTokens == nil {
		// for a forTupleExpr
		tokens = append(tokens, makeToken(hclsyntax.TokenCBrack, "]"))
	} else {
		// for a forObjectExpr
		tokens = append(tokens, makeToken(hclsyntax.TokenCBrace, "}"))
	}

	return tokens
}

func convertIndexExpr(state *convertState, inBlock bool, scopes *scopes,
	fullyQualifiedPath string, expr *hclsyntax.IndexExpr,
) hclwrite.Tokens {
	collection := convertExpression(state, inBlock, scopes, fullyQualifiedPath, expr.Collection)

	// If key is a static string and the container is an object (not a map),
	// prefer attribute access: myObject.someKey
	if name, _ := matchStaticString(expr.Key); name != nil && fullyQualifiedPath != "" {
		if isMap := scopes.isMap(fullyQualifiedPath); !(isMap != nil && *isMap) {
			keyName := *name
			if state.rewriteObjectKeys {
				fullyQualifiedPath = appendPath(fullyQualifiedPath, keyName)
				keyName = scopes.pulumiName(keyName, fullyQualifiedPath)
			}
			tokens := collection
			tokens = append(tokens, makeToken(hclsyntax.TokenDot, "."))
			tokens = append(tokens, hclwrite.TokensForIdentifier(keyName)...)
			return tokens
		}
	}

	// Fallback: bracket indexing
	key := convertExpression(state, false, scopes, "", expr.Key)
	tokens := collection
	tokens = append(tokens, makeToken(hclsyntax.TokenOBrack, "["))
	tokens = append(tokens, key...)
	tokens = append(tokens, makeToken(hclsyntax.TokenCBrack, "]"))
	return tokens
}

func convertSplatExpr(state *convertState, inBlock bool, scopes *scopes,
	fullyQualifiedPath string, expr *hclsyntax.SplatExpr,
) hclwrite.Tokens {
	source := convertExpression(state, inBlock, scopes, "", expr.Source)
	each := convertExpression(state, false, scopes, "", expr.Each)

	tokens := source
	tokens = append(tokens, makeToken(hclsyntax.TokenOBrack, "["))
	tokens = append(tokens, makeToken(hclsyntax.TokenStar, "*"))
	tokens = append(tokens, makeToken(hclsyntax.TokenCBrack, "]"))
	tokens = append(tokens, each...)
	return tokens
}

func convertAnonSymbolExpr(scopes *scopes,
	fullyQualifiedPath string, expr *hclsyntax.AnonSymbolExpr,
) hclwrite.Tokens {
	return hclwrite.Tokens{}
}

func convertConditionalExpr(state *convertState, inBlock bool, scopes *scopes,
	fullyQualifiedPath string, expr *hclsyntax.ConditionalExpr,
) hclwrite.Tokens {
	condition := convertExpression(state, inBlock, scopes, "", expr.Condition)
	trueResult := convertExpression(state, false, scopes, "", expr.TrueResult)
	falseResult := convertExpression(state, inBlock, scopes, "", expr.FalseResult)

	tokens := condition
	tokens = append(tokens, makeToken(hclsyntax.TokenQuestion, "?"))
	tokens = append(tokens, trueResult...)
	tokens = append(tokens, makeToken(hclsyntax.TokenColon, ":"))
	tokens = append(tokens, falseResult...)
	return tokens
}

func convertParenthesesExpr(state *convertState, inBlock bool, scopes *scopes,
	fullyQualifiedPath string, expr *hclsyntax.ParenthesesExpr,
) hclwrite.Tokens {
	tokens := hclwrite.Tokens{makeToken(hclsyntax.TokenOParen, "(")}
	tokens = append(tokens, convertExpression(state, false, scopes, "", expr.Expression)...)
	tokens = append(tokens, makeToken(hclsyntax.TokenCParen, ")"))
	leading, trailing := getTrivia(state.sources, expr.SrcRange, inBlock)
	return append(leading, append(tokens, trailing...)...)
}

func convertExpression(state *convertState, inBlock bool, scopes *scopes,
	fullyQualifiedPath string, expr hcl.Expression,
) hclwrite.Tokens {
	if expr == nil {
		return nil
	}

	switch expr := expr.(type) {
	case *hclsyntax.TupleConsExpr:
		return convertTupleConsExpr(state, inBlock, scopes, fullyQualifiedPath, expr)
	case *hclsyntax.ObjectConsExpr:
		return convertObjectConsExpr(state, inBlock, scopes, fullyQualifiedPath, expr)
	case *hclsyntax.ObjectConsKeyExpr:
		return convertObjectConsKeyExpr(state, inBlock, scopes, fullyQualifiedPath, expr)
	case *hclsyntax.FunctionCallExpr:
		return convertFunctionCallExpr(state, scopes, fullyQualifiedPath, expr)
	case *hclsyntax.LiteralValueExpr:
		return convertLiteralValueExpr(state, inBlock, expr)
	case *hclsyntax.TemplateExpr:
		return convertTemplateExpr(state, scopes, fullyQualifiedPath, expr)
	case *hclsyntax.TemplateJoinExpr:
		return convertTemplateJoinExpr(state, inBlock, scopes, fullyQualifiedPath, expr)
	case *hclsyntax.ScopeTraversalExpr:
		return convertScopeTraversalExpr(state, inBlock, scopes, fullyQualifiedPath, expr)
	case *hclsyntax.BinaryOpExpr:
		return convertBinaryOpExpr(state, inBlock, scopes, fullyQualifiedPath, expr)
	case *hclsyntax.UnaryOpExpr:
		return convertUnaryOpExpr(state, inBlock, scopes, fullyQualifiedPath, expr)
	case *hclsyntax.ForExpr:
		return convertForExpr(state, inBlock, scopes, fullyQualifiedPath, expr)
	case *hclsyntax.IndexExpr:
		return convertIndexExpr(state, inBlock, scopes, fullyQualifiedPath, expr)
	case *hclsyntax.RelativeTraversalExpr:
		return convertRelativeTraversalExpr(state, inBlock, scopes, fullyQualifiedPath, expr)
	case *hclsyntax.SplatExpr:
		return convertSplatExpr(state, inBlock, scopes, fullyQualifiedPath, expr)
	case *hclsyntax.AnonSymbolExpr:
		return convertAnonSymbolExpr(scopes, fullyQualifiedPath, expr)
	case *hclsyntax.TemplateWrapExpr:
		return convertTemplateWrapExpr(state, scopes, fullyQualifiedPath, expr)
	case *hclsyntax.ConditionalExpr:
		return convertConditionalExpr(state, inBlock, scopes, fullyQualifiedPath, expr)
	case *hclsyntax.ParenthesesExpr:
		return convertParenthesesExpr(state, inBlock, scopes, fullyQualifiedPath, expr)
	}
	contract.Failf("Couldn't convert expression: %T, %v", expr, expr.Range())
	return nil
}

type bodyAttrTokens struct {
	Line   int // This is so we can sort these in source order
	Name   string
	Trivia hclwrite.Tokens // Leading trivia
	Value  hclwrite.Tokens // The value _and_ trailing trivia
}

func tokensForObject(ts []bodyAttrTokens) hclwrite.Tokens {
	attrs := make([]hclwrite.ObjectAttrTokens, 0, len(ts))
	for _, attr := range ts {
		name := append(attr.Trivia, hclwrite.TokensForIdentifier(attr.Name)...)
		attrs = append(attrs, hclwrite.ObjectAttrTokens{
			Name:  name,
			Value: attr.Value,
		})
	}
	return hclwrite.TokensForObject(attrs)
}

type bodyAttrsTokens []bodyAttrTokens

func (ts bodyAttrsTokens) Len() int      { return len(ts) }
func (ts bodyAttrsTokens) Swap(i, j int) { ts[i], ts[j] = ts[j], ts[i] }
func (ts bodyAttrsTokens) Less(i, j int) bool {
	return ts[i].Line < ts[j].Line
}

func (ts bodyAttrsTokens) Line() int {
	if len(ts) == 0 {
		return 0
	}
	line := math.MaxInt32
	for _, attr := range ts {
		if attr.Line < line {
			line = attr.Line
		}
	}
	return line
}

func expressionTypePath(expr hcl.Expression) string {
	path := ""

	computePath := func(traversal hcl.Traversal) {
		for _, part := range traversal {
			switch part := part.(type) {
			case hcl.TraverseRoot:
				path = path + part.Name
			case hcl.TraverseAttr:
				path = path + "." + part.Name
			}
		}
	}

	switch expr := expr.(type) {
	case *hclsyntax.ScopeTraversalExpr:
		computePath(expr.Traversal)
	case *hclsyntax.RelativeTraversalExpr:
		computePath(expr.Traversal)
	}

	return path
}

func bodyContent(body hcl.Body) *hcl.BodyContent {
	// We want to exclude any hidden blocks and attributes, and the only way to do that with hcl.Body is to
	// give it a schema. JustAttributes() will return all non-hidden attributes, but will error if there's
	// any blocks, and there's no equivalent to get non-hidden attributes and blocks.
	hclSchema := &hcl.BodySchema{}
	// The `body` passed in here _should_ be a hclsyntax.Body. That's currently the only way to just iterate
	// all the raw blocks of a hcl.Body.
	synbody, ok := body.(*hclsyntax.Body)
	contract.Assertf(ok, "%T was not a hclsyntax.Body", body)
	for _, block := range synbody.Blocks {
		if block.Type != "dynamic" {
			hclSchema.Blocks = append(hclSchema.Blocks, hcl.BlockHeaderSchema{Type: block.Type})
		} else {
			// Dynamic blocks have labels on them, we need to tell the schema that's ok.
			hclSchema.Blocks = append(hclSchema.Blocks, hcl.BlockHeaderSchema{
				Type:       block.Type,
				LabelNames: block.Labels,
			})
		}
	}
	for _, attr := range synbody.Attributes {
		hclSchema.Attributes = append(hclSchema.Attributes, hcl.AttributeSchema{Name: attr.Name})
	}
	content, diagnostics := body.Content(hclSchema)
	contract.Assertf(len(diagnostics) == 0, "diagnostics was not empty: %s", diagnostics.Error())
	return content
}

// Convert a hcl.Body treating sub-bodies as attributes
func convertBody(state *convertState, scopes *scopes, fullyQualifiedPath string, body hcl.Body) bodyAttrsTokens {
	contract.Assertf(fullyQualifiedPath != "", "fullyQualifiedPath should not be empty")

	content := bodyContent(body)
	newAttributes := make(bodyAttrsTokens, 0)

	// If we see blocks we turn those into lists (unless maxItems==1)
	blockLists := make(map[string][]bodyAttrsTokens)
	for _, block := range content.Blocks {
		if block.Type == "timeouts" {
			// Timeouts are a special resource option block, we can't currently convert that PCL so just skip
			continue
		}

		blockPath := appendPath(fullyQualifiedPath, block.Type)
		name := block.Type
		if block.Type == "dynamic" {
			name = block.Labels[0]
			// For dynamic blocks the path is the first label, not "dynamic"
			blockPath = appendPath(fullyQualifiedPath, name)
		}
		// If this is a list so add [] to the path
		isList := !scopes.maxItemsOne(blockPath) && !scopes.isResource(blockPath)
		name = scopes.pulumiName(name, blockPath)
		if isList {
			blockPath = appendPathArray(blockPath)
		}

		if block.Type == "dynamic" {
			dynamicBody, ok := block.Body.(*hclsyntax.Body)
			contract.Assertf(ok, "%T was not a hclsyntax.Body", dynamicBody)

			// This block _might_ have an "iterator" entry to set the variable name
			var tfEachVar string
			iteratorAttr, has := dynamicBody.Attributes["iterator"]
			if has {
				str, _ := matchStaticString(iteratorAttr.Expr)
				if str == nil {
					panic("iterator must be a static string")
				}
				tfEachVar = *str
			} else {
				tfEachVar = block.Labels[0]
			}

			pulumiEachVar, cleanup := scopes.addNestedScopeUniqueName("entry", "", "")
			defer cleanup()

			dynamicTokens := hclwrite.Tokens{makeToken(hclsyntax.TokenOBrack, "[")}
			dynamicTokens = append(dynamicTokens, makeToken(hclsyntax.TokenIdent, "for"))
			dynamicTokens = append(dynamicTokens, makeToken(hclsyntax.TokenIdent, pulumiEachVar))
			dynamicTokens = append(dynamicTokens, makeToken(hclsyntax.TokenIdent, "in"))

			forEachAttr, hasForEachAttr := dynamicBody.Attributes["for_each"]
			if !hasForEachAttr {
				continue
			}

			// wrap the collection expression into `entries(collection)` so that each entry has key and value
			forEachExprTokens := convertExpression(state, true, scopes, fullyQualifiedPath, forEachAttr.Expr)
			dynamicTokens = append(dynamicTokens, makeToken(hclsyntax.TokenIdent, "entries"))
			dynamicTokens = append(dynamicTokens, makeToken(hclsyntax.TokenOParen, "("))
			dynamicTokens = append(dynamicTokens, forEachExprTokens...)
			dynamicTokens = append(dynamicTokens, makeToken(hclsyntax.TokenCParen, ")"))
			dynamicTokens = append(dynamicTokens, makeToken(hclsyntax.TokenColon, ":"))

			bodyTokens := hclwrite.Tokens{makeToken(hclsyntax.TokenIdent, "{}")}
			for _, innerBlock := range dynamicBody.Blocks {
				if innerBlock.Type == "content" {
					scopes.push(map[string]string{
						tfEachVar: pulumiEachVar,
					})
					contentBody := convertBody(state, scopes, blockPath, innerBlock.Body)
					bodyTokens = tokensForObject(contentBody)
					scopes.pop()
				}
			}

			dynamicTokens = append(dynamicTokens, bodyTokens...)
			dynamicTokens = append(dynamicTokens, makeToken(hclsyntax.TokenCBrack, "]"))

			if !isList {
				// This is a block attribute, not a list
				dynamicTokens = hclwrite.TokensForFunctionCall("singleOrNone", dynamicTokens)
			}

			newAttributes = append(newAttributes, bodyAttrTokens{
				Name:  name,
				Value: dynamicTokens,
			})
		} else {
			if !isList {
				// This is a block attribute, not a list
				newAttributes = append(newAttributes, bodyAttrTokens{
					Line:  block.DefRange.Start.Line,
					Name:  name,
					Value: tokensForObject(convertBody(state, scopes, blockPath, block.Body)),
				})
			} else {
				list := blockLists[name]
				list = append(list, convertBody(state, scopes, blockPath, block.Body))
				blockLists[name] = list
			}
		}
	}

	// Iterate the blocks we've found in sorted order, we'll resort the attributes later by line position but
	// this ensure that any state mutation (like looking up names of undeclared resources/datasources) is
	// consistent. We do the same for the attributes below as well.
	names := maps.Keys(blockLists)
	sort.Strings(names)
	for _, name := range names {
		items := blockLists[name]

		listTokens := hclwrite.Tokens{makeToken(hclsyntax.TokenOBrack, "[")}
		first := true
		line := math.MaxInt32
		for _, item := range items {
			if !first {
				listTokens = append(listTokens, makeToken(hclsyntax.TokenComma, ","))
			}
			first = false
			listTokens = append(listTokens, tokensForObject(item)...)
			if item.Line() < line {
				line = item.Line()
			}
		}
		listTokens = append(listTokens, makeToken(hclsyntax.TokenCBrack, "]"))

		newAttributes = append(newAttributes, bodyAttrTokens{
			Line:  line,
			Name:  name,
			Value: listTokens,
		})
	}

	// As above, iterate in sorted order
	names = maps.Keys(content.Attributes)
	sort.Strings(names)
	for _, name := range names {
		attr := content.Attributes[name]
		attrPath := appendPath(fullyQualifiedPath, attr.Name)

		var name string
		if state.rewriteObjectKeys {
			name = scopes.pulumiName(attr.Name, attrPath)
		} else {
			name = attr.Name
		}

		// We need the leading trivia here, but the trailing trivia will be handled by convertExpression
		leading, _ := getTrivia(state.sources, getAttributeRange(state.sources, attr.Expr.Range()), true)
		expr := convertExpression(state, true, scopes, attrPath, attr.Expr)

		// If this is a maxItemsOne property then in terraform it will be a list, but in Pulumi it will be a
		// single value. We need to project the list expression we've just converted into a single value, for
		// literals we can just take the single item from the tuple literal, for everything else we just have
		// to assume we can index at [0].
		if scopes.maxItemsOne(attrPath) {
			targetExpressionPath := expressionTypePath(attr.Expr)
			if scopes.isPropertyPath(targetExpressionPath) {
				// the attribute is being assigned to an expression which is a traversal
				// we check here whether the result of the traversal is marked with max items = 1
				// because if that the case, we shouldn't project it to singleton
				if !scopes.maxItemsOne(targetExpressionPath) {
					expr = projectListToSingleton(expr)
				}
			} else {
				expr = projectListToSingleton(expr)
			}
		}

		asset := scopes.isAsset(attrPath)
		if asset != nil {
			if asset.Kind == tfbridge.FileArchive || asset.Kind == tfbridge.BytesArchive {
				expr = hclwrite.TokensForFunctionCall("fileArchive", expr)
			} else {
				expr = hclwrite.TokensForFunctionCall("fileAsset", expr)
			}
		}

		newAttributes = append(newAttributes, bodyAttrTokens{
			Line:   attr.Range.Start.Line,
			Name:   name,
			Trivia: leading,
			Value:  expr,
		})
	}
	sort.Sort(newAttributes)
	return newAttributes
}

// camelCaseObjectAttributes rewrites the attributes of objects to camelCase and returns the modified value.
// when the input is a list of objects or map of objects, those are modified recursively.
func camelCaseObjectAttributes(value cty.Value) cty.Value {
	// if the value is null, return as is
	if value.IsNull() {
		return value
	}

	// handle type object({...})
	if value.Type().IsObjectType() && value.LengthInt() > 0 {
		modifiedAttributes := map[string]cty.Value{}
		for propertyKey, propertyValue := range value.AsValueMap() {
			modifiedValue := camelCaseObjectAttributes(propertyValue)
			modifiedAttributes[camelCaseName(propertyKey)] = modifiedValue
		}
		return cty.ObjectVal(modifiedAttributes)
	}

	// handle type list(...)
	if value.Type().IsListType() && value.LengthInt() > 0 {
		modifiedValues := make([]cty.Value, value.LengthInt())
		for index, element := range value.AsValueSlice() {
			modifiedValues[index] = camelCaseObjectAttributes(element)
		}

		return cty.ListVal(modifiedValues)
	}

	// handle set(...) and convert it to list(...)
	// because we simplify sets to lists
	if value.Type().IsSetType() && value.LengthInt() > 0 {
		modifiedValues := make([]cty.Value, value.LengthInt())
		for index, element := range value.AsValueSet().Values() {
			modifiedValues[index] = camelCaseObjectAttributes(element)
		}

		return cty.ListVal(modifiedValues)
	}

	if value.Type().IsTupleType() && value.LengthInt() > 0 {
		tupleValues := make([]cty.Value, value.LengthInt())
		for index, element := range value.AsValueSlice() {
			tupleValues[index] = camelCaseObjectAttributes(element)
		}

		return cty.TupleVal(tupleValues)
	}

	// handle type map(object({...}))
	if value.Type().IsMapType() && value.LengthInt() > 0 {
		modifiedAttributes := map[string]cty.Value{}
		for propertyKey, propertyValue := range value.AsValueMap() {
			modifiedValue := camelCaseObjectAttributes(propertyValue)
			modifiedAttributes[propertyKey] = modifiedValue
		}
		return cty.MapVal(modifiedAttributes)
	}

	// anything else, return as is
	return value
}

func convertVariable(state *convertState, scopes *scopes,
	variable *configs.Variable,
) (hclwrite.Tokens, *hclwrite.Block, hclwrite.Tokens) {
	pulumiName := scopes.roots["var."+variable.Name].Name
	labels := []string{pulumiName}

	pulumiType := convertCtyType(variable.Type)
	if !variable.Default.IsNull() && variable.Type == cty.DynamicPseudoType {
		// If we don't have an explicit type but we do have a default value, use its type
		// Only do this for primitive types. For complex types such as objects and lists
		// keep the type dynamic since it is usually used as such
		pulumiType = inferPrimitiveType(variable.Default.Type(), pulumiType)
	}

	// Don't add the "any" type explicitly, it's the default
	if pulumiType != "any" {
		labels = append(labels, pulumiType)
	}

	block := hclwrite.NewBlock("config", labels)
	blockBody := block.Body()
	if !variable.Default.IsNull() {
		// object-typed config variables rewrite their object members from snake_case to camelCase
		// for example object({ first_key = string }) becomes object({ firstKey = string })
		// so here we also rewrite the attributes of the default value to camelCase
		// i.e. { first_key = "hello" } becomes { firstKey = "hello" }
		modifiedDefault := camelCaseObjectAttributes(variable.Default)
		blockBody.SetAttributeValue("default", modifiedDefault)
	} else if variable.Default.Type() != cty.NilType {
		// default is null can mean two things:
		//  - the default attribute is not set
		//  - it is set explicitly to null
		// Here we check whether default is set to null by checking that it has a type
		// We write it out as such
		blockBody.SetAttributeValue("default", cty.NilVal)
	}

	if variable.DescriptionSet {
		blockBody.SetAttributeValue("description", cty.StringVal(variable.Description))
	}
	if variable.NullableSet {
		blockBody.SetAttributeValue("nullable", cty.BoolVal(variable.Nullable))
	}
	leading, trailing := getTrivia(state.sources, variable.DeclRange, false)
	return leading, block, trailing
}

func impliedProvider(typeName string) string {
	if under := strings.Index(typeName, "_"); under != -1 {
		typeName = typeName[:under]
	}
	return typeName
}

// Best guess at converting a tf type to a pulumi type
func impliedToken(typeName string) string {
	if under := strings.Index(typeName, "_"); under != -1 {
		provider := typeName[:under]
		typeName = typeName[under+1:]
		return fmt.Sprintf("%s:index:%s", provider, pascalCaseName(typeName))
	}

	return camelCaseName(typeName)
}

func convertLocal(state *convertState, scopes *scopes,
	local *configs.Local,
) (hclwrite.Tokens, string, hclwrite.Tokens, hclwrite.Tokens) {
	identifier := scopes.roots["local."+local.Name].Name
	expr := convertExpression(state, true, scopes, "", local.Expr)
	// The trailing trivia will have been caught by convertExpression, but we need the leading trivia before the identifier
	leading, _ := getTrivia(state.sources, local.DeclRange, true)
	return leading, identifier, expr, nil
}

func convertDataResource(state *convertState,
	info ProviderInfoSource, scopes *scopes,
	dataResource *configs.Resource,
) (hclwrite.Tokens, string, hclwrite.Tokens, hclwrite.Tokens) {
	// We translate dataResources into invokes
	path := "data." + dataResource.Type + "." + dataResource.Name
	root, has := scopes.roots[path]
	contract.Assertf(has, "data resource %s not found", dataResource.Name)
	pulumiName := root.Name

	// We special case the old template_file data resource to just return not implemented for now, eventually
	// we want to map this to a templating function in std.
	if dataResource.Type == "template_file" {
		text := cty.StringVal("The template_file data resource is not yet supported.")
		dataResourceExpression := hclwrite.TokensForFunctionCall("notImplemented", hclwrite.TokensForValue(text))
		leading, trailing := getTrivia(state.sources, dataResource.DeclRange, false)
		return leading, pulumiName, dataResourceExpression, trailing
	}

	invokeToken := cty.StringVal(impliedToken(dataResource.Type))
	if root.DataSourceInfo != nil {
		invokeToken = cty.StringVal(root.DataSourceInfo.Tok.String())
	}

	// If count is set we'll make this into an array expression
	var countExpr hclwrite.Tokens
	if dataResource.Count != nil {
		countExpr = convertExpression(state, true, scopes, "", dataResource.Count)
		scopes.countIndex = hcl.Traversal{hcl.TraverseRoot{Name: "__index"}}
	}

	// If for_each is set we'll make this into an object expression
	var forEachExpr hclwrite.Tokens
	if dataResource.ForEach != nil {
		forEachExpr = convertExpression(state, true, scopes, "", dataResource.ForEach)
		scopes.eachKey = hcl.Traversal{hcl.TraverseRoot{Name: "__key"}}
		scopes.eachValue = hcl.Traversal{hcl.TraverseRoot{Name: "__value"}}
	}

	invokeArgs := convertBody(state, scopes, path, dataResource.Config)

	functionCall := hclwrite.TokensForFunctionCall(
		"invoke",
		hclwrite.TokensForValue(invokeToken),
		tokensForObject(invokeArgs))

	dataResourceExpression := functionCall
	// If count is set then we need to turn this into a for array expression
	if dataResource.Count != nil {
		dataResourceExpression = hclwrite.Tokens{makeToken(hclsyntax.TokenOBrack, "[")}
		dataResourceExpression = append(dataResourceExpression, makeToken(hclsyntax.TokenIdent, "for"))
		dataResourceExpression = append(dataResourceExpression, makeToken(hclsyntax.TokenIdent, "__index"))
		dataResourceExpression = append(dataResourceExpression, makeToken(hclsyntax.TokenIdent, "in"))
		dataResourceExpression = append(dataResourceExpression, hclwrite.TokensForFunctionCall("range", countExpr)...)
		dataResourceExpression = append(dataResourceExpression, makeToken(hclsyntax.TokenColon, ":"))
		dataResourceExpression = append(dataResourceExpression, functionCall...)
		dataResourceExpression = append(dataResourceExpression, makeToken(hclsyntax.TokenCBrack, "]"))
	}

	// If for_each is set then we need to turn this into a for object expression
	if dataResource.ForEach != nil {
		dataResourceExpression = hclwrite.Tokens{makeToken(hclsyntax.TokenOBrace, "{")}
		dataResourceExpression = append(dataResourceExpression, makeToken(hclsyntax.TokenIdent, "for"))
		dataResourceExpression = append(dataResourceExpression, makeToken(hclsyntax.TokenIdent, "__key"))
		dataResourceExpression = append(dataResourceExpression, makeToken(hclsyntax.TokenComma, ","))
		dataResourceExpression = append(dataResourceExpression, makeToken(hclsyntax.TokenIdent, "__value"))
		dataResourceExpression = append(dataResourceExpression, makeToken(hclsyntax.TokenIdent, "in"))
		dataResourceExpression = append(dataResourceExpression, forEachExpr...)
		dataResourceExpression = append(dataResourceExpression, makeToken(hclsyntax.TokenColon, ":"))
		dataResourceExpression = append(dataResourceExpression, makeToken(hclsyntax.TokenIdent, "__key"))
		dataResourceExpression = append(dataResourceExpression, makeToken(hclsyntax.TokenFatArrow, "=>"))
		dataResourceExpression = append(dataResourceExpression, functionCall...)
		dataResourceExpression = append(dataResourceExpression, makeToken(hclsyntax.TokenCBrace, "}"))
	}

	// Clear any count and each
	scopes.countIndex = nil
	scopes.eachKey = nil
	scopes.eachValue = nil
	leading, trailing := getTrivia(state.sources, dataResource.DeclRange, false)
	return leading, pulumiName, dataResourceExpression, trailing
}

func convertProvisioner(
	state *convertState,
	info ProviderInfoSource, scopes *scopes,
	provisioner *configs.Provisioner,
	resourceName string, provisionerIndex int,
	forEach hcl.Expression,
	target *hclwrite.Body,
) {
	if provisioner.Type != "local-exec" {
		// We don't support anything other than local-exec for now
		target.AppendUnstructuredTokens(hclwrite.Tokens{
			&hclwrite.Token{
				Type:  hclsyntax.TokenComment,
				Bytes: []byte("// Unsupported provisioner type " + provisioner.Type),
			},
		})
		return
	}

	provisionerName := fmt.Sprintf("%sProvisioner%d", resourceName, provisionerIndex)

	labels := []string{provisionerName, "command:local:Command"}
	block := hclwrite.NewBlock("resource", labels)
	blockBody := block.Body()

	optionsBlock := blockBody.AppendNewBlock("options", nil)
	optionsBlockBody := optionsBlock.Body()

	if forEach != nil {
		forEachExpr := convertExpression(state, true, scopes, "", forEach)
		scopes.eachKey = hcl.Traversal{hcl.TraverseRoot{Name: "range"}, hcl.TraverseAttr{Name: "key"}}
		scopes.eachValue = hcl.Traversal{hcl.TraverseRoot{Name: "range"}, hcl.TraverseAttr{Name: "value"}}
		optionsBlockBody.SetAttributeRaw("range", forEachExpr)
	}

	// The first provisioner dependsOn the resource we're provisioning, each provisioner after that depends on
	// the previous provisioner
	var dependsOn hclwrite.Tokens
	if forEach == nil {
		dependsOn = append(dependsOn, makeToken(hclsyntax.TokenOBrack, "["))
	}

	if provisionerIndex == 0 {
		dependsOn = append(dependsOn, makeToken(hclsyntax.TokenIdent, resourceName))
	} else {
		dependsOn = append(dependsOn, makeToken(hclsyntax.TokenIdent,
			fmt.Sprintf("%sProvisioner%d", resourceName, (provisionerIndex-1))))
	}

	if forEach == nil {
		dependsOn = append(dependsOn, makeToken(hclsyntax.TokenCBrack, "]"))
	}

	optionsBlockBody.SetAttributeRaw("dependsOn", dependsOn)

	attributes, _ := provisioner.Config.JustAttributes()
	var command, interpreter, environment hclwrite.Tokens
	for _, attr := range attributes {
		if attr.Name == "command" {
			command = convertExpression(state, true, scopes, "", attr.Expr)
		}
		if attr.Name == "interpreter" {
			interpreter = convertExpression(state, true, scopes, "", attr.Expr)
		}
		if attr.Name == "environment" {
			environment = convertExpression(state, true, scopes, "", attr.Expr)
		}
	}

	onDestroy := provisioner.When == configs.ProvisionerWhenDestroy

	if onDestroy {
		// We need to set create & update to a command that does nothing, and set destroy to the actual command
		blockBody.SetAttributeValue("create", cty.StringVal("true"))
		blockBody.SetAttributeValue("update", cty.StringVal("true"))
		blockBody.SetAttributeRaw("delete", command)
	} else {
		blockBody.SetAttributeRaw("create", command)
	}

	if len(interpreter) != 0 {
		blockBody.SetAttributeRaw("interpreter", interpreter)
	}

	if len(environment) != 0 {
		blockBody.SetAttributeRaw("environment", environment)
	}

	target.AppendBlock(block)
}

func convertManagedResources(state *convertState,
	info ProviderInfoSource, scopes *scopes,
	managedResource *configs.Resource,
	target *hclwrite.Body,
) {
	// We translate managedResources into resources
	path := managedResource.Type + "." + managedResource.Name
	root, has := scopes.roots[path]
	contract.Assertf(has, "resource %s not found", path)
	pulumiName := root.Name

	resourceToken := impliedToken(managedResource.Type)
	if root.ResourceInfo != nil {
		resourceToken = root.ResourceInfo.Tok.String()
	}

	labels := []string{pulumiName, resourceToken}
	block := hclwrite.NewBlock("resource", labels)
	blockBody := block.Body()

	// If the pulumi name differs from the terraform name we should set __logicalName so that we don't change
	// the name of the resource in state.
	if pulumiName != managedResource.Name {
		logicalName := state.getOriginalNameIfRenamed(managedResource.Name)
		blockBody.SetAttributeRaw("__logicalName", hclwrite.TokensForValue(cty.StringVal(logicalName)))
	}

	var options *hclwrite.Block
	// Does this resource have dependencies? If so set the "dependsOn" attribute
	if len(managedResource.DependsOn) > 0 {
		options = hclwrite.NewBlock("options", nil)
		dependsOn := hclwrite.Tokens{makeToken(hclsyntax.TokenOBrack, "[")}
		for idx, dep := range managedResource.DependsOn {
			if idx > 0 {
				dependsOn = append(dependsOn, makeToken(hclsyntax.TokenComma, ","))
			}
			tokens := rewriteTraversal(state, scopes, "", dep)
			dependsOn = append(dependsOn, tokens...)
		}
		dependsOn = append(dependsOn, makeToken(hclsyntax.TokenCBrack, "]"))
		options.Body().SetAttributeRaw("dependsOn", dependsOn)
	}

	if managedResource.Managed != nil && managedResource.Managed.CreateBeforeDestroySet {
		state.appendDiagnostic(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "converting create_before_destroy lifecycle hook is not supported",
			Detail: `in Pulumi, resources are always created before destroy unless the resource is created with the
resource option deleteBeforeReplace. If this behavior is desired, it must be set.
See https://www.pulumi.com/docs/iac/concepts/options/deletebeforereplace/ for details`,
			Subject: managedResource.DeclRange.Ptr(),
			Context: managedResource.DeclRange.Ptr(),
		})
	}

	if len(managedResource.TriggersReplacement) > 0 {
		state.appendDiagnostic(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "converting replace_triggered_by lifecycle hook is not supported",
			Subject:  managedResource.DeclRange.Ptr(),
			Context:  managedResource.DeclRange.Ptr(),
		})
	}

	// Does this resource have a count? If so set the "range" attribute
	if managedResource.Count != nil {
		if options == nil {
			options = hclwrite.NewBlock("options", nil)
		}
		countExpr := convertExpression(state, true, scopes, "", managedResource.Count)
		// Set the count_index scope
		scopes.countIndex = hcl.Traversal{hcl.TraverseRoot{Name: "range"}, hcl.TraverseAttr{Name: "value"}}
		options.Body().SetAttributeRaw("range", countExpr)
	}
	if managedResource.ForEach != nil {
		if options == nil {
			options = hclwrite.NewBlock("options", nil)
		}
		forEachExpr := convertExpression(state, true, scopes, "", managedResource.ForEach)
		scopes.eachKey = hcl.Traversal{hcl.TraverseRoot{Name: "range"}, hcl.TraverseAttr{Name: "key"}}
		scopes.eachValue = hcl.Traversal{hcl.TraverseRoot{Name: "range"}, hcl.TraverseAttr{Name: "value"}}
		options.Body().SetAttributeRaw("range", forEachExpr)
	}

	if options != nil {
		blockBody.AppendBlock(options)
	}

	resourceArgs := convertBody(state, scopes, path, managedResource.Config)
	for _, arg := range resourceArgs {
		blockBody.SetAttributeRaw(arg.Name, arg.Value)
	}

	// Clear any index we set
	scopes.countIndex = nil
	scopes.eachKey = nil
	scopes.eachValue = nil
	leading, trailing := getTrivia(state.sources, managedResource.DeclRange, false)

	target.AppendUnstructuredTokens(leading)
	target.AppendBlock(block)
	target.AppendUnstructuredTokens(trailing)

	// Add "command:Command" resources to handle provisioners
	for idx, provisioner := range managedResource.Managed.Provisioners {
		convertProvisioner(state, info, scopes, provisioner, pulumiName, idx, managedResource.ForEach, target)
	}
}

func convertPulumiTerraformModuleCall(
	state *convertState,
	scopes *scopes,
	pulumiTerraformModule *PulumiTerraformModule,
) (hclwrite.Tokens, *hclwrite.Block, hclwrite.Tokens) {
	// We translate module calls into components
	moduleCall := pulumiTerraformModule.moduleCall
	path := "module." + moduleCall.Name
	pulumiName := scopes.roots[path].Name

	token := pulumiTerraformModule.packageName + ":index:Module"
	labels := []string{pulumiName, token}
	block := hclwrite.NewBlock("resource", labels)
	blockBody := block.Body()

	// Does this resource have a count? If so set the "range" attribute
	if moduleCall.Count != nil {
		options := blockBody.AppendNewBlock("options", nil)
		countExpr := convertExpression(state, true, scopes, "", moduleCall.Count)
		// Set the count_index scope
		scopes.countIndex = hcl.Traversal{hcl.TraverseRoot{Name: "range"}, hcl.TraverseAttr{Name: "value"}}
		options.Body().SetAttributeRaw("range", countExpr)
	}

	if moduleCall.ForEach != nil {
		options := blockBody.AppendNewBlock("options", nil)
		forEachExpr := convertExpression(state, true, scopes, "", moduleCall.ForEach)
		scopes.eachKey = hcl.Traversal{hcl.TraverseRoot{Name: "range"}, hcl.TraverseAttr{Name: "key"}}
		scopes.eachValue = hcl.Traversal{hcl.TraverseRoot{Name: "range"}, hcl.TraverseAttr{Name: "value"}}
		options.Body().SetAttributeRaw("range", forEachExpr)
	}

	state.disableRewritingObjectKeys(func() {
		// sandboxed modules keep their inputs and outputs without rewriting
		moduleArgs := convertBody(state, scopes, path, moduleCall.Config)
		for _, arg := range moduleArgs {
			blockBody.SetAttributeRaw(arg.Name, arg.Value)
		}
	})

	// Clear any index we set
	scopes.countIndex = nil
	scopes.eachKey = nil
	scopes.eachValue = nil
	leading, trailing := getTrivia(state.sources, moduleCall.DeclRange, false)
	return leading, block, trailing
}

func convertModuleCall(
	state *convertState,
	scopes *scopes,
	modules map[moduleKey]string,
	destinationDirectory string,
	moduleCall *configs.ModuleCall,
) (hclwrite.Tokens, *hclwrite.Block, hclwrite.Tokens) {
	// We translate module calls into components
	path := "module." + moduleCall.Name
	pulumiName := scopes.roots[path].Name

	// Get the local component path from the module source
	moduleKey := makeModuleKey(moduleCall)
	modulePath, has := modules[moduleKey]
	if !has {
		// This is a genuine system panic, we shoudn't ever hit this.
		panic("module not found")
	}
	// modulePath will always be rooted, but we want these paths to show as relative in the .pp files so we
	// need the relative path from the current destination directory
	relPath, err := filepath.Rel(destinationDirectory, modulePath)
	if err != nil {
		// This is a genuine system panic, we shoudn't ever hit this because we made the modulePaths relative
		// to destinationDirectory earlier.
		panic(fmt.Sprintf("failed to get relative path from %s to %s: %v", destinationDirectory, modulePath, err))
	}
	// relPath will be an OS path, but we want to consistently write out unix style paths
	relPath = filepath.ToSlash(relPath)

	// Rel will have cleaned the path, but we want to preserve the ./ prefix (unless it's already got a ../ prefix)
	if !strings.HasPrefix(relPath, "../") {
		relPath = "./" + relPath
	}
	labels := []string{pulumiName, relPath}
	block := hclwrite.NewBlock("component", labels)
	blockBody := block.Body()

	// Does this resource have a count? If so set the "range" attribute
	if moduleCall.Count != nil {
		options := blockBody.AppendNewBlock("options", nil)
		countExpr := convertExpression(state, true, scopes, "", moduleCall.Count)
		// Set the count_index scope
		scopes.countIndex = hcl.Traversal{hcl.TraverseRoot{Name: "range"}, hcl.TraverseAttr{Name: "value"}}
		options.Body().SetAttributeRaw("range", countExpr)
	}

	if moduleCall.ForEach != nil {
		options := blockBody.AppendNewBlock("options", nil)
		forEachExpr := convertExpression(state, true, scopes, "", moduleCall.ForEach)
		scopes.eachKey = hcl.Traversal{hcl.TraverseRoot{Name: "range"}, hcl.TraverseAttr{Name: "key"}}
		scopes.eachValue = hcl.Traversal{hcl.TraverseRoot{Name: "range"}, hcl.TraverseAttr{Name: "value"}}
		options.Body().SetAttributeRaw("range", forEachExpr)
	}

	moduleArgs := convertBody(state, scopes, path, moduleCall.Config)
	for _, arg := range moduleArgs {
		blockBody.SetAttributeRaw(arg.Name, arg.Value)
	}

	// Clear any index we set
	scopes.countIndex = nil
	scopes.eachKey = nil
	scopes.eachValue = nil
	leading, trailing := getTrivia(state.sources, moduleCall.DeclRange, false)
	return leading, block, trailing
}

func convertOutput(state *convertState, scopes *scopes,
	output *configs.Output,
) (hclwrite.Tokens, *hclwrite.Block, hclwrite.Tokens) {
	labels := []string{scopes.roots["output."+output.Name].Name}
	block := hclwrite.NewBlock("output", labels)
	blockBody := block.Body()
	leading, _ := getTrivia(state.sources, getAttributeRange(state.sources, output.Expr.Range()), true)
	blockBody.AppendUnstructuredTokens(leading)
	blockBody.SetAttributeRaw("value", convertExpression(state, true, scopes, "", output.Expr))

	leading, trailing := getTrivia(state.sources, output.DeclRange, false)
	return leading, block, trailing
}

// An "item" from a terraform file
type terraformItem struct {
	variable   *configs.Variable
	local      *configs.Local
	data       *configs.Resource
	moduleCall *configs.ModuleCall
	resource   *configs.Resource
	output     *configs.Output
	provider   *configs.Provider
}

func (item terraformItem) DeclRange() hcl.Range {
	if item.variable != nil {
		return item.variable.DeclRange
	}
	if item.local != nil {
		return item.local.DeclRange
	}
	if item.data != nil {
		return item.data.DeclRange
	}
	if item.resource != nil {
		return item.resource.DeclRange
	}
	if item.moduleCall != nil {
		return item.moduleCall.DeclRange
	}
	if item.output != nil {
		return item.output.DeclRange
	}
	if item.provider != nil {
		return item.provider.DeclRange
	}
	panic("at least one of the fields in terraformItem should be set!")
}

type terraformItems []terraformItem

func (items terraformItems) Len() int      { return len(items) }
func (items terraformItems) Swap(i, j int) { items[i], items[j] = items[j], items[i] }
func (items terraformItems) Less(i, j int) bool {
	a := items[i].DeclRange()
	b := items[j].DeclRange()

	if a.Filename < b.Filename {
		return true
	} else if a.Filename > b.Filename {
		return false
	}
	return a.Start.Line < b.Start.Line
}

// Used to key into the modules map for the given address and version.
type moduleKey struct {
	source  addrs.ModuleSource
	version string
}

func (key moduleKey) WithSource(source addrs.ModuleSource) moduleKey {
	return moduleKey{
		source:  source,
		version: key.version,
	}
}

func (key moduleKey) WithVersion(version string) moduleKey {
	return moduleKey{
		source:  key.source,
		version: version,
	}
}

func makeModuleKey(call *configs.ModuleCall) moduleKey {
	return moduleKey{
		source:  call.SourceAddr,
		version: call.Version.Required.String(),
	}
}

func translateRemoteModule(
	modules map[moduleKey]string, // A map of module source addresses to paths in destination.
	packageAddr string, // The address of the remote terraform module to translate.
	packageSubdir string,
	destinationRoot afero.Fs, // The root of the destination filesystem to write PCL to.
	destinationDirectory string, // A path in destination to write the translated code to.
	info ProviderInfoSource,
	requiredProviders map[string]*configs.RequiredProvider,
	sandboxedModules map[string]*PulumiTerraformModule,
	generatedProjectDirectory string,
) hcl.Diagnostics {
	fetcher := getmodules.NewPackageFetcher()
	tempPath, err := os.MkdirTemp("", "pulumi-tf-registry")
	if err != nil {
		return hcl.Diagnostics{
			&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Failed to create temporary directory",
				Detail:   fmt.Sprintf("Failed to create temporary directory to download module: %v", err),
			},
		}
	}
	instPath := filepath.Join(tempPath, "src")

	err = fetcher.FetchPackage(context.TODO(), instPath, packageAddr)
	if err != nil {
		return hcl.Diagnostics{
			&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Failed to download module",
				Detail:   fmt.Sprintf("Failed to download module: %v", err),
			},
		}
	}

	modDir, err := getmodules.ExpandSubdirGlobs(instPath, packageSubdir)
	if err != nil {
		return hcl.Diagnostics{
			&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Failed to expand module subdirectory",
				Detail:   fmt.Sprintf("Failed to expand module subdirectory: %v", err),
			},
		}
	}

	sourceRoot := afero.NewBasePathFs(afero.NewOsFs(), modDir)

	return translateModuleSourceCode(
		modules,
		sourceRoot, "/",
		destinationRoot, destinationDirectory,
		info,
		requiredProviders,
		sandboxedModules,
		/*topLevelModule*/ false,
		generatedProjectDirectory,
	)
}

func findPulumiTerraformModuleAnnotation(trivia string) (string, bool) {
	lines := strings.Split(trivia, "\n")
	for _, line := range lines {
		parts := strings.Split(strings.ReplaceAll(line, "//", ""), " ")
		partsLength := len(parts)
		for i, part := range parts {
			if part == pulumiTerraformModuleAnnotation && i < partsLength-1 {
				packageName := parts[i+1]
				return packageName, true
			}
		}
	}

	return "", false
}

func moduleFromRemoteRegistry(moduleCall *configs.ModuleCall) (addrs.ModuleSourceRegistry, bool) {
	switch source := moduleCall.SourceAddr.(type) {
	case addrs.ModuleSourceRegistry:
		return source, true
	default:
		return addrs.ModuleSourceRegistry{}, false
	}
}

func moduleFromLocalFileSystem(moduleCall *configs.ModuleCall) (addrs.ModuleSourceLocal, bool) {
	switch source := moduleCall.SourceAddr.(type) {
	case addrs.ModuleSourceLocal:
		return source, true
	default:
		return "", false
	}
}

// resolveRemoteRegistryModule resolves the module source and its concrete version from the remote registry
// based on the module call.
//
// TODO: this duplicates logic from pulumi-terraform-module provider and we could probably delegate the work to it.
// Possibly using `pulumi package get-schema terraform-module <source>` and
// extracting the version from the resulting schema
func resolveRemoteRegistryModule(moduleCall *configs.ModuleCall) (addrs.ModuleSourceRegistry, *version.Version, error) {
	empty := addrs.ModuleSourceRegistry{}
	source, isRemote := moduleFromRemoteRegistry(moduleCall)
	if !isRemote {
		return empty, nil, fmt.Errorf("module source for %s is not from a remote registry", moduleCall.Name)
	}

	services := disco.NewWithCredentialsSource(nil)
	reg := registry.NewClient(services, nil)
	regsrcAddr := regsrc.ModuleFromRegistryPackageAddr(source.Package)
	resp, err := reg.ModuleVersions(context.TODO(), regsrcAddr)
	if err != nil {
		return empty, nil, fmt.Errorf("failed to retrieve available versions for %s: %s", source, err)
	}
	modMeta := resp.Modules[0]
	var latestVersion *version.Version
	for _, mv := range modMeta.Versions {
		v, err := version.NewVersion(mv.Version)
		if err != nil {
			return empty, nil, fmt.Errorf("failed to parse version %q for %s: %s", mv.Version, source, err)
		}
		if v.Prerelease() != "" {
			continue
		}
		if (latestVersion == nil || v.GreaterThan(latestVersion)) && moduleCall.Version.Required.Check(v) {
			latestVersion = v
		}
	}

	if latestVersion == nil {
		return empty, nil, fmt.Errorf("failed to find version for %s that matched %s",
			source,
			moduleCall.Version.Required)
	}

	return source, latestVersion, nil
}

func translateModuleSourceCode(
	modules map[moduleKey]string, // A map of module source addresses to paths in destination.
	sourceRoot afero.Fs, // The root of the source terraform package.
	sourceDirectory string, // The path in sourceRoot to the source terraform module.
	destinationRoot afero.Fs, // The root of the destination filesystem to write PCL to.
	destinationDirectory string, // A path in destination to write the translated code to.
	info ProviderInfoSource,
	requiredProviders map[string]*configs.RequiredProvider,
	sandboxedModules map[string]*PulumiTerraformModule,
	topLevelModule bool,
	generatedProjectDirectory string,
) hcl.Diagnostics {
	sources, module, moduleDiagnostics := loadConfigDir(sourceRoot, sourceDirectory)
	if moduleDiagnostics.HasErrors() {
		// No syntax.Files to return here because we're relying on terraform to load and parse, means no
		// source context gets printed with warnings/errors here.
		return moduleDiagnostics
	}

	for name, provider := range module.ProviderRequirements.RequiredProviders {
		requiredProviders[name] = provider
	}

	scopes := newScopes()

	state := &convertState{
		sources:              sources,
		diagnostics:          hcl.Diagnostics{},
		rewriteObjectKeys:    true,
		sandboxedModuleNames: make(map[string]string),
		renames: pclOverlapRenames{
			nameToRename: make(map[string]string),
			renameToName: make(map[string]string),
		},
		typeRenames: pclOverlapRenames{
			nameToRename: make(map[string]string),
			renameToName: make(map[string]string),
		},
	}

	// First go through and add everything to the items list so we can sort it by source order
	items := make(terraformItems, 0)
	for _, variable := range module.Variables {
		_, name := state.renamePclOverlap("variable", nil, variable.Name, &variable.DeclRange)
		variable.Name = name
		items = append(items, terraformItem{variable: variable})
	}
	for _, local := range module.Locals {
		_, name := state.renamePclOverlap("local", nil, local.Name, &local.DeclRange)
		local.Name = name
		items = append(items, terraformItem{local: local})
	}
	for _, data := range module.DataResources {
		typeName, name := state.renamePclOverlap("data", &data.Type, data.Name, &data.DeclRange)
		data.Type = typeName
		data.Name = name
		items = append(items, terraformItem{data: data})
	}
	for _, moduleCall := range module.ModuleCalls {
		_, name := state.renamePclOverlap("moduleCall", nil, moduleCall.Name, &moduleCall.DeclRange)
		moduleCall.Name = name
		items = append(items, terraformItem{moduleCall: moduleCall})
	}
	for _, resource := range module.ManagedResources {
		// TODO do all types and diagnostics.
		typeName, name := state.renamePclOverlap("resource", &resource.Type, resource.Name, &resource.DeclRange)
		resource.Type = typeName
		resource.Name = name
		items = append(items, terraformItem{resource: resource})
	}
	for _, output := range module.Outputs {
		_, name := state.renamePclOverlap("output", nil, output.Name, &output.DeclRange)
		output.Name = name
		items = append(items, terraformItem{output: output})
	}
	for _, provider := range module.ProviderConfigs {
		_, name := state.renamePclOverlap("provider", nil, provider.Name, &provider.DeclRange)
		provider.Name = name
		items = append(items, terraformItem{provider: provider})
	}
	// Now sort that items array by source location
	sort.Sort(items)

	// Now go through and generate unique names for all the things
	for _, item := range items {
		if item.variable != nil {
			scopes.getOrAddPulumiName("var."+item.variable.Name, "", "Config")
		}
	}
	for _, item := range items {
		if item.local != nil {
			key := "local." + item.local.Name
			scopes.getOrAddPulumiName(key, "my", "")
			root := scopes.roots[key]
			root.Expression = &item.local.Expr
			scopes.roots[key] = root
		}
	}
	for _, item := range items {
		if item.data != nil {
			dataResource := item.data
			key := "data." + dataResource.Type + "." + dataResource.Name
			// Try to grab the info for this data type
			provider := impliedProvider(dataResource.Type)
			root := PathInfo{}
			if provider != "template" {
				// We rewrite uses of template because it's really common but the provider for it is
				// deprecated. As such we don't want to try and do a mapping lookup for it.

				providerInfo, err := info.GetProviderInfo(provider, requiredProviders[provider])
				if err != nil {
					state.appendDiagnostic(&hcl.Diagnostic{
						Subject:  &dataResource.DeclRange,
						Severity: hcl.DiagWarning,
						Summary:  "Failed to get provider info",
						Detail:   fmt.Sprintf("Failed to get provider info for %q: %v", dataResource.Type, err),
					})
				}

				if providerInfo != nil {
					root.Resource = providerInfo.P.DataSourcesMap().Get(dataResource.Type)
					root.DataSourceInfo = providerInfo.DataSources[dataResource.Type]
				}
			}

			invokeToken := impliedToken(dataResource.Type)
			if root.DataSourceInfo != nil {
				invokeToken = root.DataSourceInfo.Tok.String()
			}
			tokenParts := strings.Split(invokeToken, ":")
			//nolint:staticcheck
			suffix := strings.Title(tokenParts[len(tokenParts)-1])
			root.Name = scopes.getOrAddPulumiName(key, "", suffix)
			scopes.roots[key] = root
		}
	}
	for _, item := range items {
		if item.resource != nil {
			managedResource := item.resource
			key := managedResource.Type + "." + managedResource.Name
			// Try to grab the info for this resource type
			provider := impliedProvider(managedResource.Type)
			providerInfo, err := info.GetProviderInfo(provider, requiredProviders[provider])
			if err != nil {
				state.appendDiagnostic(&hcl.Diagnostic{
					Subject:  &managedResource.DeclRange,
					Severity: hcl.DiagWarning,
					Summary:  "Failed to get provider info",
					Detail:   fmt.Sprintf("Failed to get provider info for %q: %v", managedResource.Type, err),
				})
			}

			root := PathInfo{}
			if providerInfo != nil {
				root.Resource = providerInfo.P.ResourcesMap().Get(managedResource.Type)
				root.ResourceInfo = providerInfo.Resources[managedResource.Type]
			}

			resourceToken := impliedToken(managedResource.Type)
			if root.ResourceInfo != nil {
				resourceToken = root.ResourceInfo.Tok.String()
			}
			tokenParts := strings.Split(resourceToken, ":")
			//nolint:staticcheck
			suffix := strings.Title(tokenParts[len(tokenParts)-1])
			root.Name = scopes.getOrAddPulumiName(key, "", suffix)
			scopes.roots[key] = root
		}
	}
	for _, item := range items {
		if item.moduleCall != nil {
			leadingTrivia, _ := getTrivia(sources, item.moduleCall.DeclRange, false)
			packageName, wrapUsingTerraformModule := findPulumiTerraformModuleAnnotation(string(leadingTrivia.Bytes()))
			moduleCall := item.moduleCall
			moduleName := scopes.getOrAddPulumiName("module."+moduleCall.Name, "", "Component")
			moduleKey := makeModuleKey(moduleCall)

			if !wrapUsingTerraformModule {
				// check if other modules have been marked with @sandbox with the same source
				// in that case, we consider this module as sandboxed too
				for _, sandboxModule := range sandboxedModules {
					if sandboxModule.moduleCall.SourceAddr == moduleCall.SourceAddr {
						wrapUsingTerraformModule = true
						packageName = sandboxModule.packageName
						break
					}
				}
			}

			if source, isRemote := moduleFromRemoteRegistry(moduleCall); isRemote {
				key := fmt.Sprintf("%s-%s-%s", moduleCall.Name, source.Package.Namespace, source.Package.Name)
				if _, seen := sandboxedModules[key]; !seen && wrapUsingTerraformModule {
					state.sandboxedModuleNames[moduleCall.Name] = moduleName
					sandboxedModules[key] = &PulumiTerraformModule{
						packageName: packageName,
						moduleCall:  moduleCall,
					}
				}
			}

			if source, isLocal := moduleFromLocalFileSystem(moduleCall); isLocal {
				key := filepath.Clean(filepath.Join(sourceDirectory, string(source)))
				if _, seen := sandboxedModules[key]; !seen && wrapUsingTerraformModule {
					state.sandboxedModuleNames[moduleCall.Name] = moduleName
					sandboxedModules[key] = &PulumiTerraformModule{
						packageName:     packageName,
						moduleCall:      moduleCall,
						local:           true,
						absolutePath:    key,
						targetDirectory: generatedProjectDirectory,
						copyModule:      false,
					}
				}
			}

			// check if this module has been seen before and it shouldn't be wrapped.
			// If it has, we don't need to translate it again.
			if _, hasBeenSeen := modules[moduleKey]; !hasBeenSeen && !wrapUsingTerraformModule {
				// We need the source code for this module. But it might be a reference to a module from the
				// registry (e.g. "terraform-aws-modules/s3-bucket/aws")
				addr := moduleCall.SourceAddr
				switch addr := addr.(type) {
				case addrs.ModuleSourceLocal:
					// Local modules are the simplest case, the module is in the same package just at a
					// different path. We need to do another check for uniquness here though as multiple
					// terraform modules may refer to the same destination module but via different relative
					// paths. When we store the module in the modules map we'll store the relative path, but
					// also the absolute path to allow this lookup to hit later.
					absoluteAddr := addrs.ModuleSourceLocal(
						filepath.Clean(filepath.Join(sourceDirectory, string(addr))))
					absoluteKey := moduleKey.WithSource(absoluteAddr)
					if destinationPath, has := modules[absoluteKey]; has {
						// We've already seen this module, just save this new relative address
						modules[moduleKey] = destinationPath
						continue
					}

					sourcePath := filepath.Join(sourceDirectory, addr.String())
					destinationPath := filepath.Join(destinationDirectory, addr.String())
					// Check that this path isn't already taken
					for _, path := range modules {
						if path == destinationPath {
							// We ought to do better than this, and try and find a uniuqe path but just
							// erroring for now is fine.
							state.appendDiagnostic(&hcl.Diagnostic{
								Severity: hcl.DiagError,
								Summary:  "Duplicate module path",
								Detail:   fmt.Sprintf("The module path %q is already taken by another module", destinationPath),
							})
							return state.diagnostics
						}
					}
					modules[moduleKey] = destinationPath
					modules[absoluteKey] = destinationPath

					diags := translateModuleSourceCode(
						modules,
						sourceRoot,
						sourcePath,
						destinationRoot,
						destinationPath,
						info,
						requiredProviders,
						sandboxedModules,
						/*topLevelModule*/ false,
						generatedProjectDirectory,
					)
					state.diagnostics = append(state.diagnostics, diags...)
					if diags.HasErrors() {
						return state.diagnostics
					}

				case addrs.ModuleSourceRemote:
					// Get the _name_ of this module, which is the last part of the path
					moduleName := filepath.Base(addr.String())
					destinationPath := filepath.Join(destinationDirectory, moduleName)
					// Check that this path isn't already taken
					for _, path := range modules {
						if path == destinationPath {
							// We ought to do better than this, and try and find a uniuqe path but just
							// erroring for now is fine.
							state.appendDiagnostic(&hcl.Diagnostic{
								Severity: hcl.DiagError,
								Summary:  "Duplicate module path",
								Detail:   fmt.Sprintf("The module path %q is already taken by another module", destinationPath),
							})
							return state.diagnostics
						}
					}
					modules[moduleKey] = destinationPath

					diags := translateRemoteModule(
						modules,
						addr.Package.String(),
						addr.Subdir,
						destinationRoot,
						destinationPath,
						info,
						requiredProviders,
						sandboxedModules,
						generatedProjectDirectory,
					)
					state.diagnostics = append(state.diagnostics, diags...)
					if diags.HasErrors() {
						return state.diagnostics
					}

				case addrs.ModuleSourceRegistry:
					// Similar to ModuleSourceRemote but we have to use the registry client to get the go-getter address.
					services := disco.NewWithCredentialsSource(nil)
					reg := registry.NewClient(services, nil)
					regsrcAddr := regsrc.ModuleFromRegistryPackageAddr(addr.Package)
					resp, err := reg.ModuleVersions(context.TODO(), regsrcAddr)
					if err != nil {
						state.appendDiagnostic(&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Error accessing remote module registry",
							Detail:   fmt.Sprintf("Failed to retrieve available versions for %s: %s", addr, err),
						})
						return state.diagnostics
					}
					modMeta := resp.Modules[0]
					var latestVersion *version.Version
					for _, mv := range modMeta.Versions {
						v, err := version.NewVersion(mv.Version)
						if err != nil {
							state.appendDiagnostic(&hcl.Diagnostic{
								Severity: hcl.DiagError,
								Summary:  "Error accessing remote module registry",
								Detail:   fmt.Sprintf("Failed to parse version %q for %s: %s", mv.Version, addr, err),
							})
							return state.diagnostics
						}
						if v.Prerelease() != "" {
							continue
						}
						if (latestVersion == nil || v.GreaterThan(latestVersion)) && moduleCall.Version.Required.Check(v) {
							latestVersion = v
						}
					}

					if latestVersion == nil {
						state.appendDiagnostic(&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Error accessing remote module registry",
							Detail:   fmt.Sprintf("Failed to find version for %s that matched %s", addr, moduleCall.Version.Required),
						})
						return state.diagnostics
					}

					// At this point we have a real version, we're going to resolve our module key to _that_ version and look it
					// up again
					absoluteKey := moduleKey.WithVersion(latestVersion.String())
					if _, has := modules[absoluteKey]; has {
						// We've seen this before, we can just use that path
						modules[moduleKey] = modules[absoluteKey]
						continue
					}

					realAddrRaw, err := reg.ModuleLocation(context.TODO(), regsrcAddr, latestVersion.String())
					if err != nil {
						state.appendDiagnostic(&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Error accessing remote module registry",
							Detail:   fmt.Sprintf("Failed to retrieve a download URL for %s %s: %s", addr, latestVersion, err),
						})
						return state.diagnostics
					}
					realAddr, err := addrs.ParseModuleSource(realAddrRaw)
					if err != nil {
						state.appendDiagnostic(&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Invalid package location from module registry",
							Detail: fmt.Sprintf(
								"Module registry returned invalid source location %q for %s %s: %s.",
								realAddrRaw, addr, latestVersion, err),
						})
						return state.diagnostics
					}
					var remoteAddr addrs.ModuleSourceRemote
					switch realAddr := realAddr.(type) {
					// Only a remote source address is allowed here: a registry isn't
					// allowed to return a local path (because it doesn't know what
					// its being called from) and we also don't allow recursively pointing
					// at another registry source for simplicity's sake.
					case addrs.ModuleSourceRemote:
						remoteAddr = realAddr
					default:
						state.appendDiagnostic(&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Invalid package location from module registry",
							Detail: fmt.Sprintf(
								"Module registry returned invalid source location %q for %s %s: "+
									"must be a direct remote package address.",
								realAddrRaw, addr, latestVersion),
						})
						return state.diagnostics
					}
					// Maintain the subdir from the original module call.
					remoteAddr.Subdir = addr.Subdir

					destinationPath := filepath.Join(destinationDirectory,
						fmt.Sprintf("%s_%s", addr.Package.Name, latestVersion), addr.Subdir)
					// Check that this path isn't already taken
					for _, path := range modules {
						if path == destinationPath {
							// We ought to do better than this, and try and find a uniuqe path but just
							// erroring for now is fine.
							state.appendDiagnostic(&hcl.Diagnostic{
								Severity: hcl.DiagError,
								Summary:  "Duplicate module path",
								Detail:   fmt.Sprintf("The module path %q is already taken by another module", destinationPath),
							})
							return state.diagnostics
						}
					}
					// Update both our original key, and the resolved absolute key to point to this created path
					modules[moduleKey] = destinationPath
					modules[absoluteKey] = destinationPath

					diags := translateRemoteModule(
						modules,
						remoteAddr.Package.String(),
						remoteAddr.Subdir,
						destinationRoot,
						destinationPath,
						info,
						requiredProviders,
						sandboxedModules,
						generatedProjectDirectory,
					)
					state.diagnostics = append(state.diagnostics, diags...)
					if diags.HasErrors() {
						return state.diagnostics
					}
				}
			}
		}
	}

	for _, item := range items {
		if item.output != nil {
			scopes.getOrAddOutput("output." + item.output.Name)
		}
	}

	var pulumiYaml *workspace.Project
	for _, item := range items {
		if item.provider != nil {
			provider := item.provider
			cfgName := provider.Name
			if rename, ok := pulumiRenamedProviderNames[provider.Name]; ok {
				cfgName = rename
			}

			// If an alias is set just warn and ignore this, we can't support this yet
			if provider.Alias != "" {
				state.appendDiagnostic(&hcl.Diagnostic{
					Subject:  &provider.DeclRange,
					Severity: hcl.DiagWarning,
					Summary:  "Provider alias not supported",
					Detail:   fmt.Sprintf("Provider aliases are not supported, ignoring %s=%s", provider.Name, provider.Alias),
				})
				continue
			}

			// Set the project name to the folder name
			if pulumiYaml == nil {
				projectName := filepath.Base(sourceDirectory)
				pulumiYaml = &workspace.Project{
					Name: tokens.PackageName(projectName),
					// We _have_ to fill in a runtime here because otherwise the CLI errors when loading the
					// Pulumi.yaml, even though it will just overwrite this.
					Runtime: workspace.NewProjectRuntimeInfo("terraform", nil),
				}
			}

			// Try to grab the info for this provider config
			providerInfo, err := info.GetProviderInfo(provider.Name, requiredProviders[provider.Name])
			if err != nil {
				state.appendDiagnostic(&hcl.Diagnostic{
					Subject:  &provider.DeclRange,
					Severity: hcl.DiagWarning,
					Summary:  "Failed to get provider info",
					Detail:   fmt.Sprintf("Failed to get provider info for %q: %v", provider.Name, err),
				})
			}

			// Translate the config from this provider block to pulumi config
			if pulumiYaml.Config == nil {
				pulumiYaml.Config = make(map[string]workspace.ProjectConfigType)
			}
			cfg := pulumiYaml.Config

			content := bodyContent(provider.Config)

			// There might be blocks for "dynamic" or just object attributes, for now we just warn that they're being skipped
			for _, block := range content.Blocks {
				state.appendDiagnostic(&hcl.Diagnostic{
					Subject:  &block.DefRange,
					Severity: hcl.DiagWarning,
					Summary:  "Provider config not supported",
					Detail:   fmt.Sprintf("Blocks in provider config are not supported, ignoring %s:%s", provider.Name, block.Type),
				})
			}

			// We need to iterate over the attributes in a stable order to ensure we get the same output
			attrKeys := make([]string, 0, len(content.Attributes))
			for name := range content.Attributes {
				attrKeys = append(attrKeys, name)
			}
			sort.Slice(attrKeys, func(i, j int) bool {
				ia := content.Attributes[attrKeys[i]]
				ja := content.Attributes[attrKeys[j]]

				return ia.Range.Start.Line < ja.Range.Start.Line
			})

			for _, attrKey := range attrKeys {
				// Evauluate and marshal the attribute to a YAML like value for Pulumi config
				value := content.Attributes[attrKey]
				val, diags := scopes.EvalExpr(value.Expr)
				if diags.HasErrors() {
					state.appendDiagnostic(&hcl.Diagnostic{
						Subject:  &provider.DeclRange,
						Severity: hcl.DiagWarning,
						Summary:  "Failed to evaluate provider config",
						Detail:   fmt.Sprintf("Could not evaluate expression for %s:%s", cfgName, attrKey),
					})
					// If we couldn't eval the config we'll emit an obvious TODO to the config for it
					val = cty.StringVal("TODO: " + state.sourceCode(value.Expr.Range()))
				}

				// Simplest way to get a cty type into YAML is to roundtrip it through JSON
				buffer, err := json.Marshal(ctyjson.SimpleJSONValue{Value: val})
				if err != nil {
					state.appendDiagnostic(&hcl.Diagnostic{
						Subject:  &provider.DeclRange,
						Severity: hcl.DiagError,
						Summary:  "Failed to marshal provider config",
						Detail:   fmt.Sprintf("Could not marshal value for %s:%s: %v", provider.Name, attrKey, err),
					})
					continue
				}
				var yamlValue interface{}
				err = json.Unmarshal(buffer, &yamlValue)
				if err != nil {
					state.appendDiagnostic(&hcl.Diagnostic{
						Subject:  &provider.DeclRange,
						Severity: hcl.DiagError,
						Summary:  "Failed to marshal provider config",
						Detail:   fmt.Sprintf("Failed to unmarshal provider config for %s:%s: %v", provider.Name, attrKey, err),
					})
					continue
				}

				// Check if we need to rename this config key, but default to camelcase
				name := camelCaseName(attrKey)
				if providerInfo != nil {
					if info, has := providerInfo.Config[attrKey]; has && info.Name != "" {
						name = info.Name
					}
				}

				// When there is a renamed provider be sure to update it's name in the Pulumi.yaml config.
				cfg[cfgName+":"+name] = workspace.ProjectConfigType{
					Value: yamlValue,
				}
			}
		}
	}

	pclFiles := make(map[string]*hclwrite.File)

	// We want to write things out to matching .pp files and in source order
	for _, item := range items {
		r := item.DeclRange()
		path := changeExtension(r.Filename, ".pp")
		path, err := filepath.Rel(sourceDirectory, path)
		if err != nil {
			panic("Rel should never fail")
		}
		file := pclFiles[path]
		if file == nil {
			file = hclwrite.NewFile()
			pclFiles[path] = file
		}

		body := file.Body()

		// Add package metadata to only the top level module.
		// This should be true for only the entry point into the conversion, all
		// recursive module translations should set this false.
		// This guarantees that the package blocks are only written in one place (eg main.pp)
		if topLevelModule {
			// Deterministic ordering.
			providerBlockNames := make([]string, 0, len(requiredProviders))
			for name := range requiredProviders {
				providerBlockNames = append(providerBlockNames, name)
			}
			slices.Sort(providerBlockNames)

			for _, name := range providerBlockNames {
				provider := requiredProviders[name]
				block, d := getPackageBlock(name, provider)
				state.diagnostics = append(state.diagnostics, d...)
				body.AppendBlock(block)
				body.AppendUnstructuredTokens(hclwrite.TokensForIdentifier("\n"))
			}

			moduleKeys := make([]string, 0, len(sandboxedModules))
			for key := range sandboxedModules {
				moduleKeys = append(moduleKeys, key)
			}
			slices.Sort(providerBlockNames)

			declaredPackages := codegen.NewStringSet()

			for _, key := range moduleKeys {
				sandboxModule := sandboxedModules[key]
				packageName := sandboxModule.packageName
				if declaredPackages.Has(packageName) {
					continue
				}
				declaredPackages.Add(packageName)
				if !sandboxModule.local {
					source, version, err := resolveRemoteRegistryModule(sandboxModule.moduleCall)
					if err != nil {
						state.diagnostics = append(state.diagnostics,
							errorf(sandboxModule.moduleCall.DeclRange, "resolving remote module: %s", err.Error()))
						continue
					}

					modulePackegeBlock := remotePulumiTerraformModulePackageBlock(sandboxModule.packageName, source, version.String())
					body.AppendBlock(modulePackegeBlock)
					body.AppendUnstructuredTokens(hclwrite.TokensForIdentifier("\n"))
				} else {
					relativePath, err := filepath.Rel(sandboxModule.targetDirectory, sandboxModule.absolutePath)
					contract.AssertNoErrorf(err, "failed to get relative path for %s", sandboxModule.absolutePath)
					modulePackageBlock := localPulumiTerraformModulePackageBlock(sandboxModule.packageName, relativePath)
					body.AppendBlock(modulePackageBlock)
					body.AppendUnstructuredTokens(hclwrite.TokensForIdentifier("\n"))
				}
			}

			// Only write the package block once
			topLevelModule = false
		}

		// First handle any inputs, these will be picked up by the "vars" scope
		if item.variable != nil {
			leading, block, trailing := convertVariable(state, scopes, item.variable)
			body.AppendUnstructuredTokens(leading)
			body.AppendBlock(block)
			body.AppendUnstructuredTokens(trailing)
		}
		// Next handle any locals, these will be picked up by the "locals" scope
		if item.local != nil {
			leading, name, value, trailing := convertLocal(state, scopes, item.local)
			body.AppendUnstructuredTokens(leading)
			body.SetAttributeRaw(name, value)
			body.AppendUnstructuredTokens(trailing)
		}
		// Next handle any data sources
		if item.data != nil {
			leading, name, value, trailing := convertDataResource(state, info, scopes, item.data)
			body.AppendUnstructuredTokens(leading)
			body.SetAttributeRaw(name, value)
			body.AppendUnstructuredTokens(trailing)
		}
		// Next handle any resources
		if item.resource != nil {
			convertManagedResources(state, info, scopes, item.resource, body)
		}
		// Next handle any modules
		if item.moduleCall != nil {
			if source, isRemote := moduleFromRemoteRegistry(item.moduleCall); isRemote {
				key := fmt.Sprintf("%s-%s-%s", item.moduleCall.Name, source.Package.Namespace, source.Package.Name)
				if sandboxedModule, ok := sandboxedModules[key]; ok {
					_, block, trailing := convertPulumiTerraformModuleCall(state, scopes, sandboxedModule)
					body.AppendBlock(block)
					body.AppendUnstructuredTokens(trailing)
				} else {
					leading, block, trailing := convertModuleCall(state, scopes, modules, destinationDirectory, item.moduleCall)
					body.AppendUnstructuredTokens(leading)
					body.AppendBlock(block)
					body.AppendUnstructuredTokens(trailing)
				}
			} else if source, isLocal := moduleFromLocalFileSystem(item.moduleCall); isLocal {
				key := filepath.Clean(filepath.Join(sourceDirectory, string(source)))
				if sandboxedModule, ok := sandboxedModules[key]; ok {
					_, block, trailing := convertPulumiTerraformModuleCall(state, scopes, sandboxedModule)
					body.AppendBlock(block)
					body.AppendUnstructuredTokens(trailing)
				} else {
					leading, block, trailing := convertModuleCall(state, scopes, modules, destinationDirectory, item.moduleCall)
					body.AppendUnstructuredTokens(leading)
					body.AppendBlock(block)
					body.AppendUnstructuredTokens(trailing)
				}
			} else {
				leading, block, trailing := convertModuleCall(state, scopes, modules, destinationDirectory, item.moduleCall)
				body.AppendUnstructuredTokens(leading)
				body.AppendBlock(block)
				body.AppendUnstructuredTokens(trailing)
			}
		}
		// Finally handle any outputs
		if item.output != nil {
			leading, block, trailing := convertOutput(state, scopes, item.output)
			body.AppendUnstructuredTokens(leading)
			body.AppendBlock(block)
			body.AppendUnstructuredTokens(trailing)
		}
	}

	// Now we've written everything generate formatted output files.
	// Always, make sure the destination directory exists even if we have nothing to write.
	err := destinationRoot.MkdirAll(destinationDirectory, 0o755)
	if err != nil {
		state.appendDiagnostic(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("could not create destination directory for pcl: %s", err),
		})
		return state.diagnostics
	}
	for key, file := range pclFiles {
		buffer := &bytes.Buffer{}
		_, err := file.WriteTo(buffer)
		if err != nil {
			state.appendDiagnostic(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("could not write pcl to memory buffer: %s", err),
			})
			return state.diagnostics
		}

		fullpath := filepath.Join(destinationDirectory, key)
		keyDirectory := filepath.Dir(fullpath)
		err = destinationRoot.MkdirAll(keyDirectory, 0o755)
		if err != nil {
			state.appendDiagnostic(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("could not create destination directory for pcl: %s", err),
			})
			return state.diagnostics
		}

		// Reformat to canonical style
		formatted := hclwrite.Format(buffer.Bytes())
		err = afero.WriteFile(destinationRoot, fullpath, formatted, 0o644)
		if err != nil {
			state.appendDiagnostic(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("could not write pcl to destination: %s", err),
			})
			return state.diagnostics
		}
	}

	// Finally write out the Pulumi.yaml file if needed
	if pulumiYaml != nil {
		fullpath := filepath.Join(destinationDirectory, "Pulumi.yaml")
		keyDirectory := filepath.Dir(fullpath)
		err := destinationRoot.MkdirAll(keyDirectory, 0o755)
		if err != nil {
			state.appendDiagnostic(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("could not create destination directory for project YAML: %s", err),
			})
			return state.diagnostics
		}

		formatted, err := yaml.Marshal(pulumiYaml)
		if err != nil {
			state.appendDiagnostic(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("could not format project YAML: %s", err),
			})
			return state.diagnostics
		}

		err = afero.WriteFile(destinationRoot, fullpath, formatted, 0o644)
		if err != nil {
			state.appendDiagnostic(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("could not write project YAML to destination: %s", err),
			})
			return state.diagnostics
		}
	}

	return state.diagnostics
}

// getPackageBlock returns package block in the following form:
//
//	package <name> {
//	  baseProviderName = <name>
//	  baseProviderVersion = <version>
//	  baseProviderDownloadUrl = <url>
//	  parameterization {
//	    name = <name>
//	    version = <version>
//	    value = <base64-encoded-value>
//	  }
//	}
func getPackageBlock(name string, prov *configs.RequiredProvider) (*hclwrite.Block, hcl.Diagnostics) {
	// if the name is one of our known providers just write it, if it is not it
	// must be tf so write a paramaterized package.

	packageNameParts := strings.Split(prov.Source, "/")
	packageName := packageNameParts[len(packageNameParts)-1]

	// Some Terraform providers correspond to Pulumi packages which have different names (e.g. "google" is actually backed
	// by the Pulumi "gcp" package). Perform that renaming now before generating definitions/looking up whether we'll
	// bridge or not.
	if renamed, ok := pulumiRenamedProviderNames[packageName]; ok {
		packageName = renamed
	}

	// TODO(pulumi/pulumi#17933) For now we just the package name portion of the source (also known as the "type" in tf
	// parlance). This may lead to name overlap, but as of now this is how our system works. If we need to fix name
	// overlap, this is the place to start.
	_ = name
	block := hclwrite.NewBlock("package", []string{packageName})
	body := block.Body()

	diags := hcl.Diagnostics{}

	if isTerraformProvider(packageName) {
		body.SetAttributeValue("baseProviderName", cty.StringVal("terraform-provider"))
		body.SetAttributeValue("baseProviderVersion", cty.StringVal("0.8.1"))

		// Right now we use the shim  of the opentofu implementation of getting the
		// TF Package version to access an internal API.
		// This is a bit of a hack and a future option may be to depend on the TF
		// bridge to provide this functionality.
		var desiredVersion versions.Version
		desiredVersion, diags = shim.FindTfPackageVersion(prov)

		// Right now this json string is just a remote source with url and version,
		// but it could be more complex in the future.
		// Once it becomes more complex it may be worthwhile to export a function
		// from pulumi-terraform-bridge that encodes this value and use it here.
		innerValue := fmt.Sprintf(`{"remote":{"url":"%s","version":"%s"}}`,
			prov.Source,
			desiredVersion.String())
		encoded := base64.StdEncoding.EncodeToString([]byte(innerValue))

		paramBlock := hclwrite.NewBlock("parameterization", []string{})
		body.AppendBlock(paramBlock)
		paramBlockBody := paramBlock.Body()
		paramBlockBody.SetAttributeValue("version", cty.StringVal(desiredVersion.String()))
		paramBlockBody.SetAttributeValue("name", cty.StringVal(packageName))
		paramBlockBody.SetAttributeValue("value", cty.StringVal(encoded))
	} else {
		body.SetAttributeValue("baseProviderName", cty.StringVal(packageName))
	}

	return block, diags
}

func remotePulumiTerraformModulePackageBlock(
	packageName string,
	source addrs.ModuleSourceRegistry,
	version string,
) *hclwrite.Block {
	block := hclwrite.NewBlock("package", []string{packageName})
	body := block.Body()
	body.SetAttributeValue("baseProviderName", cty.StringVal("terraform-module"))
	body.SetAttributeValue("baseProviderVersion", cty.StringVal("0.1.4"))
	paramBlock := hclwrite.NewBlock("parameterization", []string{})
	body.AppendBlock(paramBlock)
	paramBlockBody := paramBlock.Body()
	paramBlockBody.SetAttributeValue("name", cty.StringVal(packageName))
	paramBlockBody.SetAttributeValue("version", cty.StringVal(version))
	appendComment(paramBlockBody, "encoded parameterization values:")
	appendComment(paramBlockBody, "module: "+source.String())
	appendComment(paramBlockBody, "version: "+version)
	appendComment(paramBlockBody, "packageName: "+packageName)
	parameterization, _ := json.Marshal(map[string]string{
		"module":      source.String(),
		"version":     version,
		"packageName": packageName,
	})
	paramBlockBody.SetAttributeValue("value",
		cty.StringVal(base64.StdEncoding.EncodeToString(parameterization)))
	return block
}

func appendComment(body *hclwrite.Body, comment string) {
	line := fmt.Sprintf("// %s\n", comment)
	body.AppendUnstructuredTokens(hclwrite.Tokens{
		{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte(line),
		},
	})
}

func localPulumiTerraformModulePackageBlock(
	packageName string,
	localPath string,
) *hclwrite.Block {
	block := hclwrite.NewBlock("package", []string{packageName})
	body := block.Body()
	body.SetAttributeValue("baseProviderName", cty.StringVal("terraform-module"))
	body.SetAttributeValue("baseProviderVersion", cty.StringVal("0.1.4"))
	paramBlock := hclwrite.NewBlock("parameterization", []string{})

	// hardcoded version to align with the version in pulumi-terraform-module
	// for local modules
	packageVersion := "0.0.1"
	body.AppendBlock(paramBlock)
	paramBlockBody := paramBlock.Body()
	paramBlockBody.SetAttributeValue("name", cty.StringVal(packageName))
	paramBlockBody.SetAttributeValue("version", cty.StringVal(packageVersion))
	parameterization, _ := json.Marshal(map[string]string{
		"module":      localPath,
		"packageName": packageName,
	})
	appendComment(paramBlockBody, "encoded parameterization values:")
	appendComment(paramBlockBody, "module: "+localPath)
	appendComment(paramBlockBody, "packageName: "+packageName)
	paramBlockBody.SetAttributeValue("value",
		cty.StringVal(base64.StdEncoding.EncodeToString(parameterization)))

	return block
}

func TranslateModule(
	source afero.Fs, sourceDirectory string,
	destination afero.Fs,
	info ProviderInfoSource,
	generatedProjectDirectory string,
) hcl.Diagnostics {
	modules := make(map[moduleKey]string)
	return translateModuleSourceCode(modules,
		source,
		sourceDirectory,
		destination,
		"/",
		info,
		/*requiredProviders*/ map[string]*configs.RequiredProvider{},
		/*sanboxedModules*/ map[string]*PulumiTerraformModule{},
		/*topLevelModule*/ true,
		generatedProjectDirectory,
	)
}

func errorf(subject hcl.Range, f string, args ...interface{}) *hcl.Diagnostic {
	return diagf(hcl.DiagError, subject, f, args...)
}

func diagf(severity hcl.DiagnosticSeverity, subject hcl.Range, f string, args ...interface{}) *hcl.Diagnostic {
	message := fmt.Sprintf(f, args...)
	return &hcl.Diagnostic{
		Severity: severity,
		Summary:  message,
		Subject:  &subject,
	}
}

var unimplementedFunctionBugs = map[string]string{
	"formatdate":   "pulumi/pulumi-converter-terraform#196",
	"templatefile": "pulumi/pulumi-converter-terraform#192",
	"try":          "pulumi/pulumi-converter-terraform#16",
	"yamlencode":   "pulumi/pulumi-converter-terraform#190",
}

// getTrackingBug returns the tracking bug for the given function call, if any.
func getTrackingBug(call *hclsyntax.FunctionCallExpr) string {
	r, ok := unimplementedFunctionBugs[call.Name]
	if !ok {
		return "pulumi/pulumi-converter-terraform#65 (catch all bug)"
	}

	return r
}
