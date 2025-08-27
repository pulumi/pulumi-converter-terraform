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
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfbridge"
	shim "github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfshim"
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfshim/schema"
	"github.com/pulumi/pulumi/pkg/v3/codegen/cgstrings"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/contract"
	"github.com/pulumi/terraform/pkg/addrs"
	"github.com/pulumi/terraform/pkg/lang"
	"github.com/pulumi/terraform/pkg/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// Used to return info about a path in the schema.
type PathInfo struct {
	// The final part of the path (e.g. a_field)
	Name string

	// The Resource that contains the path (e.g. data.simple_data_source)
	Resource shim.Resource
	// The DataSourceInfo for the path (e.g. data.simple_data_source)
	DataSourceInfo *tfbridge.DataSourceInfo
	// The ResourceInfo for the path (e.g. simple_resource)
	ResourceInfo *tfbridge.ResourceInfo

	// The Schema for the path (e.g. data.simple_data_source.a_field)
	Schema shim.Schema
	// The SchemaInfo for the path (e.g. data.simple_data_source.a_field)
	SchemaInfo *tfbridge.SchemaInfo

	// The expression for a local variable
	Expression *hcl.Expression
}

type scopes struct {
	// All known roots, keyed by fully qualified path e.g. data.some_data_source
	roots map[string]PathInfo

	// Local variables that are in scope from for expressions
	locals []map[string]string

	// Set non-nil if "count.index" can be mapped
	countIndex hcl.Traversal
	eachKey    hcl.Traversal
	eachValue  hcl.Traversal

	scope *lang.Scope
}

func newScopes() *scopes {
	s := &scopes{
		roots:  make(map[string]PathInfo),
		locals: make([]map[string]string, 0),
	}
	scope := &lang.Scope{
		Data:     s,
		PureOnly: true,
		BaseDir:  ".",
	}
	s.scope = scope
	return s
}

// lookup the given name in roots and locals
func (s *scopes) lookup(name string) string {
	for i := len(s.locals) - 1; i >= 0; i-- {
		if s.locals[i][name] != "" {
			return s.locals[i][name]
		}
	}
	if root, has := s.roots[name]; has {
		return root.Name
	}
	return ""
}

func (s *scopes) push(locals map[string]string) {
	s.locals = append(s.locals, locals)
}

func (s *scopes) pop() {
	s.locals = s.locals[0 : len(s.locals)-1]
}

// isUsed returns if _any_ root scope currently uses the name "name"
func (s *scopes) isUsed(name string) bool {
	// We don't have many, but there's a few _keywords_ in pcl that are easier if we just never emit them
	if name == "range" {
		return true
	}

	for _, usedName := range s.roots {
		if usedName.Name == name {
			return true
		}
	}
	return false
}

// generateUniqueName takes "name" and ensures it's unique.
// First by appending `suffix` to it, and then appending an incrementing count
func (s *scopes) generateUniqueName(name, prefix, suffix string) string {
	// Not used, just return it
	if !s.isUsed(name) {
		return name
	}
	// It's used, so add the prefix and suffix
	if prefix != "" {
		name = prefix + cgstrings.UppercaseFirst(name)
	}

	if suffix != "" {
		name = name + cgstrings.UppercaseFirst(suffix)
	}

	if !s.isUsed(name) {
		return name
	}
	// Still used add a counter
	baseName := name
	counter := 2
	for {
		name = fmt.Sprintf("%s%d", baseName, counter)
		if !s.isUsed(name) {
			return name
		}
		counter = counter + 1
	}
}

// addNestedScopeUniqueName adds a name to the current scope, making it unique
// if needed.  Returns a function to cleanup any root modifications.
func (s *scopes) addNestedScopeUniqueName(name, prefix, suffix string) (string, func()) {
	addAndReturn := func(name string) (string, func()) {
		cleanup := func() { s.pop() }
		s.push(map[string]string{name: name})
		return name, cleanup
	}

	isUsedInAnyScope := func(name string) bool {
		isUsed := s.isUsed(name)
		if isUsed {
			return true
		}

		for _, locals := range s.locals {
			if locals[name] != "" {
				return true
			}
		}

		return false
	}

	if !isUsedInAnyScope(name) {
		return addAndReturn(name)
	}

	// It's used, so add the prefix and suffix
	if prefix != "" {
		name = prefix + cgstrings.UppercaseFirst(name)
	}

	if suffix != "" {
		name = name + cgstrings.UppercaseFirst(suffix)
	}

	// Still used add a counter
	baseName := name
	counter := 2
	for {
		name = fmt.Sprintf("%s%d", baseName, counter)
		if !isUsedInAnyScope(name) {
			return addAndReturn(name)
		}
		counter = counter + 1
	}
}

func (s *scopes) getOrAddOutput(name string) string {
	root, has := s.roots[name]
	if has {
		return root.Name
	}
	parts := strings.Split(name, ".")
	tfName := parts[len(parts)-1]
	pulumiName := camelCaseName(tfName)
	s.roots[name] = PathInfo{Name: pulumiName}
	return pulumiName
}

// getOrAddPulumiName takes "path" and returns the unique name for it. First by prepending `prefix` and
// appending `suffix` to it, and then appending an incrementing count.
func (s *scopes) getOrAddPulumiName(path, prefix, suffix string) string {
	root, has := s.roots[path]
	if has {
		return root.Name
	}
	parts := strings.Split(path, ".")
	tfName := parts[len(parts)-1]
	pulumiName := camelCaseName(tfName)
	pulumiName = s.generateUniqueName(pulumiName, prefix, suffix)
	s.roots[path] = PathInfo{Name: pulumiName}
	return pulumiName
}

// Given a fully typed path (e.g. data.simple_data_source.a_field) returns the final part of that path
// (a_field) and the either the Resource or Schema, and SchemaInfo for that path (if any).
//
// Can return (PathInfo{}, false) if the final part cannot be resolved.
// The caller is then responsible for handling that case.
func (s *scopes) getInfo(fullyQualifiedPath string) (PathInfo, bool) {
	parts := strings.Split(fullyQualifiedPath, ".")
	contract.Assertf(len(parts) >= 2, "empty path passed into getInfo: %s", fullyQualifiedPath)
	contract.Assertf(parts[0] != "", "empty path part passed into getInfo: %s", fullyQualifiedPath)
	contract.Assertf(parts[1] != "", "empty path part passed into getInfo: %s", fullyQualifiedPath)

	var getInner func(sch shim.SchemaMap, info map[string]*tfbridge.SchemaInfo, parts []string) (PathInfo, bool)
	getInner = func(sch shim.SchemaMap, info map[string]*tfbridge.SchemaInfo, parts []string) (PathInfo, bool) {
		contract.Assertf(parts[0] != "", "empty path part passed into getInfo")

		// At this point parts[0] may be an property + indexer or just a property. Work that out first.
		part, rest, indexer := strings.Cut(parts[0], "[]")

		// Lookup the info for this part
		var curSch shim.Schema
		if sch != nil {
			curSch = sch.Get(part)
		}
		curInfo := info[part]

		// We want this part
		if len(parts) == 1 && !indexer {
			return PathInfo{
				Name:       part,
				Schema:     curSch,
				SchemaInfo: curInfo,
			}, true
		}

		// Else recurse into the next part of the type, how we do this depends on if this was indexed or not
		if !indexer {
			// No indexers, simple recurse on fields
			var nextSchema shim.SchemaMap
			var nextInfo map[string]*tfbridge.SchemaInfo
			if curSch != nil {
				if sch, ok := curSch.Elem().(shim.Resource); ok {
					nextSchema = sch.Schema()
				}
			}
			if curInfo != nil {
				nextInfo = curInfo.Fields
			}
			return getInner(nextSchema, nextInfo, parts[1:])
		}

		// part was indexed (i.e. something like "part[]" or part[]foo[][]bar), so rather than looking at the
		// fields we need to look at the elements.
		var resourceOrSchema interface{}
		if curSch != nil {
			resourceOrSchema = curSch.Elem()
		}
		if curInfo != nil {
			curInfo = curInfo.Elem
		}

		if rest == "" && len(parts) == 1 {
			// This element is what we we're looking for
			res, _ := resourceOrSchema.(shim.Resource)
			sch, _ := resourceOrSchema.(shim.Schema)
			return PathInfo{
				Name:       part,
				Resource:   res,
				Schema:     sch,
				SchemaInfo: curInfo,
			}, true
		} else if rest == "" {
			// Can recurse into the next set of fields
			var nextSchema shim.SchemaMap
			var nextInfo map[string]*tfbridge.SchemaInfo
			if sch, ok := resourceOrSchema.(shim.Resource); ok {
				nextSchema = sch.Schema()
			}
			if curInfo != nil {
				nextInfo = curInfo.Fields
			}
			return getInner(nextSchema, nextInfo, parts[1:])
		}

		// Otherwise we have a complex indexer part that we can't handle yet.
		// The caller is responsible for handling this, e.g. using the name as is or
		// camelCasing it, etc.
		return PathInfo{}, false
	}

	if parts[0] == "data" {
		contract.Assertf(len(parts) >= 3, "empty path passed into getInfo: %s", fullyQualifiedPath)
		contract.Assertf(parts[2] != "", "empty path part passed into getInfo: %s", fullyQualifiedPath)

		root, has := s.roots[parts[0]+"."+parts[1]+"."+parts[2]]
		if len(parts) == 3 {
			if has {
				return root, true
			}
			// If we don't have a root, just return the name
			return PathInfo{Name: parts[2]}, true
		}

		var currentSchema shim.SchemaMap
		var currentInfo map[string]*tfbridge.SchemaInfo
		if root.Resource != nil {
			currentSchema = root.Resource.Schema()
		}
		if root.DataSourceInfo != nil {
			currentInfo = root.DataSourceInfo.Fields
		}

		return getInner(currentSchema, currentInfo, parts[3:])
	}

	root, has := s.roots[parts[0]+"."+parts[1]]

	if len(parts) == 2 {
		if has {
			return root, true
		}
		// If we don't have a root, just return the name
		return PathInfo{Name: parts[1]}, true
	}

	var currentSchema shim.SchemaMap
	var currentInfo map[string]*tfbridge.SchemaInfo
	if root.Resource != nil {
		currentSchema = root.Resource.Schema()
	}
	if root.ResourceInfo != nil {
		currentInfo = root.ResourceInfo.Fields
	}

	return getInner(currentSchema, currentInfo, parts[2:])
}

// Given a fully typed path (e.g. data.simple_data_source.my_data.a_field) returns the pulumi name for that path.
func (s *scopes) pulumiName(name, fullyQualifiedPath string) string {
	info, ok := s.getInfo(fullyQualifiedPath)
	// If we can't resolved the name then fallback to camelCasing the provided name (e.g. the Terraform name)
	if !ok {
		return camelCaseName(name)
	}

	// This should only be called for attribute paths, so panic if this returned a resource
	contract.Assertf(info.ResourceInfo == nil, "pulumiName must not be called on a resource or data source")
	contract.Assertf(info.DataSourceInfo == nil, "pulumiName must not be called on a resource or data source")

	// If we have a SchemaInfo and name use it
	schemaInfo := info.SchemaInfo
	if schemaInfo != nil && schemaInfo.Name != "" {
		return schemaInfo.Name
	}

	// If we have a shim schema use it to translate
	sch := info.Schema
	if sch != nil {
		return tfbridge.TerraformToPulumiNameV2(info.Name,
			schema.SchemaMap(map[string]shim.Schema{info.Name: sch}),
			map[string]*tfbridge.SchemaInfo{info.Name: schemaInfo})
	}

	// Else just return the name camel cased
	return camelCaseName(info.Name)
}

// Given a fully typed path (e.g. data.simple_data_source.my_data.a_field) returns if the schema says it's a map.
func (s *scopes) isMap(fullyQualifiedPath string) *bool {
	info, ok := s.getInfo(fullyQualifiedPath)
	if !ok {
		return nil
	}

	// This should only be called for attribute paths, so panic if this returned a resource
	contract.Assertf(info.ResourceInfo == nil, "isMap must not be called on a resource or data source")
	contract.Assertf(info.DataSourceInfo == nil, "isMap must not be called on a resource or data source")

	// If this is a resource it's not a map
	if s.isResource(fullyQualifiedPath) {
		isMap := false
		return &isMap
	}

	// If we have a shim schema use the type from that
	sch := info.Schema
	if sch != nil {
		isMap := sch.Type() == shim.TypeMap
		return &isMap
	}
	return nil
}

// Given a fully typed path (e.g. data.simple_data_source.a_field) returns whether a_field is a resource object.
func (s *scopes) isResource(fullyQualifiedPath string) bool {
	info, ok := s.getInfo(fullyQualifiedPath)
	if !ok {
		return false
	}

	// This should only be called for attribute paths, so panic if this returned a resource
	contract.Assertf(info.ResourceInfo == nil, "isResource must not be called on a resource or data source")
	contract.Assertf(info.DataSourceInfo == nil, "isResource must not be called on a resource or data source")

	// If we have a shim schema use its MaxItems and Type
	sch := info.Schema
	if sch != nil {
		// If it's a map of resources then return true. Map of resource is used in TF schema to represent a
		// sub-object, rather than a map of objects.
		elem := sch.Elem()
		if _, isResource := elem.(shim.Resource); sch.Type() == shim.TypeMap && isResource {
			return true
		}
	}

	// If we have a Resource schema this must be an object
	if info.Resource != nil {
		return true
	}

	return false
}

// Given a fully typed path (e.g. data.simple_data_source.a_field) returns whether a_field has maxItemsOne set
func (s *scopes) maxItemsOne(fullyQualifiedPath string) bool {
	info, ok := s.getInfo(fullyQualifiedPath)
	if !ok {
		return false
	}

	// This should only be called for attribute paths, so panic if this returned a resource
	contract.Assertf(info.ResourceInfo == nil, "maxItemsOne must not be called on a resource or data source")
	contract.Assertf(info.DataSourceInfo == nil, "maxItemsOne must not be called on a resource or data source")

	// If we have a SchemaInfo and a MaxItems override use it
	schemaInfo := info.SchemaInfo
	if schemaInfo != nil && schemaInfo.MaxItemsOne != nil {
		return *schemaInfo.MaxItemsOne
	}

	// If we have a shim schema use it's MaxItems and Type
	sch := info.Schema
	if sch != nil {
		// If this is a list or set, check if it has a maxItems of 1.
		if sch.Type() == shim.TypeList || sch.Type() == shim.TypeSet {
			return sch.MaxItems() == 1
		}
	}

	// Else assume false
	return false
}

// Given a fully typed path (e.g. data.simple_data_source.a_field) returns whether a_field has Asset information set
func (s *scopes) isAsset(fullyQualifiedPath string) *tfbridge.AssetTranslation {
	info, ok := s.getInfo(fullyQualifiedPath)
	if !ok {
		return nil
	}

	// This should only be called for attribute paths, so panic if this returned a resource
	contract.Assertf(info.ResourceInfo == nil, "isAsset must not be called on a resource or data source")
	contract.Assertf(info.DataSourceInfo == nil, "isAsset must not be called on a resource or data source")

	// If we have a SchemaInfo and a asset info return that
	schemaInfo := info.SchemaInfo
	if schemaInfo != nil {
		return schemaInfo.Asset
	}

	return nil
}

// Helper function to call into the terraform evaluator
func (s *scopes) EvalExpr(expr hcl.Expression) (cty.Value, tfdiags.Diagnostics) {
	return s.scope.EvalExpr(expr, cty.DynamicPseudoType)
}

type diagnostic struct {
	severity tfdiags.Severity
	summary  string
	subject  *tfdiags.SourceRange
}

func (d diagnostic) Severity() tfdiags.Severity {
	return d.severity
}

func (d diagnostic) Description() tfdiags.Description {
	return tfdiags.Description{
		Summary: d.summary,
	}
}

func (d diagnostic) Source() tfdiags.Source {
	return tfdiags.Source{
		Subject: d.subject,
	}
}

func (d diagnostic) FromExpr() *tfdiags.FromExpr {
	return nil
}

func (d diagnostic) ExtraInfo() interface{} {
	return nil
}

func makeErrorDiagnostic(summary string, subject tfdiags.SourceRange) tfdiags.Diagnostics {
	return tfdiags.Diagnostics{diagnostic{
		severity: tfdiags.Error,
		summary:  summary,
		subject:  &subject,
	}}
}

// We implement a minimal subset of terraform/lang.Data so we can evaluate some fixed expressions.

func (s *scopes) StaticValidateReferences(refs []*addrs.Reference, self addrs.Referenceable) tfdiags.Diagnostics {
	return nil
}

func (s *scopes) GetCountAttr(_ addrs.CountAttr, src tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return cty.NilVal, makeErrorDiagnostic("GetCountAttr is not supported", src)
}

func (s *scopes) GetForEachAttr(_ addrs.ForEachAttr, src tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return cty.NilVal, makeErrorDiagnostic("GetForEachAttr is not supported", src)
}

func (s *scopes) GetResource(_ addrs.Resource, src tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return cty.NilVal, makeErrorDiagnostic("GetResource is not supported", src)
}

func (s *scopes) GetLocalValue(addr addrs.LocalValue, src tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	// try and find the local and evaluate it
	var found *PathInfo
	for name, root := range s.roots {
		r := root
		if name == addr.String() {
			found = &r
			break
		}
	}
	if found == nil {
		return cty.NilVal, makeErrorDiagnostic("local not found", src)
	}

	val, diags := s.scope.EvalExpr(*found.Expression, cty.DynamicPseudoType)

	return val, diags
}

func (s *scopes) GetModule(_ addrs.ModuleCall, src tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return cty.NilVal, makeErrorDiagnostic("GetCountAttr is not supported", src)
}

func (s *scopes) GetPathAttr(_ addrs.PathAttr, src tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return cty.NilVal, makeErrorDiagnostic("GetPathAttr is not supported", src)
}

func (s *scopes) GetTerraformAttr(_ addrs.TerraformAttr, src tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return cty.NilVal, makeErrorDiagnostic("GetTerraformAttr is not supported", src)
}

func (s *scopes) GetInputVariable(_ addrs.InputVariable, src tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	return cty.NilVal, makeErrorDiagnostic("GetInputVariable is not supported", src)
}
