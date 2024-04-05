package convert

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/pulumi/terraform/pkg/configs"
	"github.com/spf13/afero"
)

// Provides data for [AutoFill]
type AutoFiller interface {
	// Returns a suggested automatically filled example HCL code for a given resource or data source name. If this
	// block is not supported or has no plausible examples, returns an empty string.
	AutoFill(token, name string) string

	// Returns true if the given resource or data source token can be passed to AutoFill.
	CanAutoFill(token string) bool
}

// Examines an HCL example code snippet to find dangling references to resources or data source calls. When processing
// documentation it is frequently the case that resources are implied but not listed in the original code. If such a
// reference is encountered, this consults autoFiller for a possible canonical definition and augments the program.
func AutoFill(autoFiller AutoFiller, hcl string) (string, error) {
	var buf bytes.Buffer
	fs := afero.NewMemMapFs()

	// Create a new file with some content.
	err := afero.WriteFile(fs, "infra.tf", []byte(hcl), 0600)
	if err != nil {
		return "", err
	}

	path := "."
	p := configs.NewParser(fs)
	mod, diags := p.LoadConfigDir(path)
	if diags.Errs() != nil {
		return "", errors.Join(diags.Errs()...)
	}

	v := newVisitor()
	for _, mr := range mod.ManagedResources {
		v.visitManagedResource(mr)
	}

	fmt.Fprintf(&buf, "%s\n", hcl)

	for _, dr := range v.dangling() {
		tok := dr.Token()
		if !autoFiller.CanAutoFill(tok) {
			continue
		}
		extra := autoFiller.AutoFill(tok, dr.Name())
		fmt.Fprintf(&buf, "\n%s\n", extra)
	}

	return buf.String(), nil
}

type ref string

func (x ref) Token() string {
	return strings.Split(string(x), ":::")[0]
}

func (x ref) Name() string {
	return strings.Split(string(x), ":::")[1]
}

func newRef(token, name string) ref {
	return ref(fmt.Sprintf("%s:::%s", token, name))
}

type visitor struct {
	defined    map[ref]struct{}
	referenced map[ref]struct{}
}

func newVisitor() *visitor {
	return &visitor{
		defined:    map[ref]struct{}{},
		referenced: map[ref]struct{}{},
	}
}

func (v *visitor) dangling() []ref {
	d := []ref{}
	for x := range v.referenced {
		_, isDef := v.defined[x]
		if !isDef {
			d = append(d, x)
		}
	}
	sort.Slice(d, func(i, j int) bool {
		return string(d[i]) < string(d[j])
	})
	return d
}

func (v *visitor) visitManagedResource(res *configs.Resource) {
	v.defined[newRef(res.Type, res.Name)] = struct{}{}
	v.visitBody(res.Config)
	v.visitExpr(res.Count)
	v.visitExpr(res.ForEach)
	v.visitTraversals(res.DependsOn)
	v.visitExprs(res.TriggersReplacement)
}

func (v *visitor) visitTraversal(t hcl.Traversal) {
	if len(t) < 2 {
		return
	}
	root, ok := t[0].(hcl.TraverseRoot)
	if !ok {
		return
	}
	attr, ok := t[1].(hcl.TraverseAttr)
	if !ok {
		return
	}
	v.referenced[newRef(root.Name, attr.Name)] = struct{}{}
}

func (v *visitor) visitTraversals(ts []hcl.Traversal) {
	for _, t := range ts {
		v.visitTraversal(t)
	}
}

func (v *visitor) visitAttribute(a *hcl.Attribute) {
	v.visitExpr(a.Expr)
}

func (v *visitor) visitExpr(expr hcl.Expression) {
	if expr == nil {
		return
	}
	for _, t := range expr.Variables() {
		v.visitTraversal(t)
	}
}

func (v *visitor) visitExprs(exprs []hcl.Expression) {
	for _, e := range exprs {
		v.visitExpr(e)
	}
}

func (v *visitor) visitBlock(b *hcl.Block) {
	v.visitBody(b.Body)
}

func (v *visitor) visitBody(b hcl.Body) {
	bc := bodyContent(b)
	for _, blk := range bc.Blocks {
		v.visitBlock(blk)
	}
	for _, attr := range bc.Attributes {
		v.visitAttribute(attr)
	}
}
