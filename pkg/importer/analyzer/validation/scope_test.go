package validation_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/transform/tree/treetesting"
	"github.com/google/nomos/pkg/importer/analyzer/validation"
	vt "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	ft "github.com/google/nomos/pkg/importer/filesystem/testing"
	"github.com/google/nomos/pkg/util/discovery"
	"k8s.io/apimachinery/pkg/runtime"
)

func withPath(o runtime.Object, path string) ast.FileObject {
	return ast.NewFileObject(o, cmpath.FromSlash(path))
}

func withScope(r *ast.Root) *ast.Root {
	apiInfos, err := discovery.NewAPIInfo(ft.TestAPIResourceList(ft.TestDynamicResources()))
	if err != nil {
		panic("testdata error")
	}
	discovery.AddAPIInfo(r, apiInfos)
	return r
}

func TestScope(t *testing.T) {
	var scopeTestcases = vt.MutatingVisitorTestcases{
		VisitorCtor: func() ast.Visitor {
			return validation.NewScope()
		},
		Options: func() []cmp.Option {
			return []cmp.Option{
				cmp.AllowUnexported(ast.FileObject{}),
			}
		},
		Testcases: []vt.MutatingVisitorTestcase{
			{
				Name:       "empty",
				Input:      withScope(vt.Helper.EmptyRoot()),
				ExpectNoop: true,
			},
			{
				Name:       "acme",
				Input:      withScope(vt.Helper.AcmeRoot()),
				ExpectNoop: true,
			},
			{
				Name:      "cluster resource at namespace scope",
				Input:     withScope(treetesting.BuildTree(t, withPath(vt.Helper.NomosAdminClusterRole(), "namespaces/cr.yaml"))),
				ExpectErr: true,
			},
			{
				Name:       "cluster resource at cluster scope",
				Input:      withScope(treetesting.BuildTree(t, withPath(vt.Helper.NomosAdminClusterRole(), "cluster/cr.yaml"))),
				ExpectNoop: true,
			},
			{
				Name:      "namespace resource at cluster scope",
				Input:     withScope(treetesting.BuildTree(t, withPath(vt.Helper.AdminRoleBinding(), "cluster/cr.yaml"))),
				ExpectErr: true,
			},
			{
				Name:       "namespace resource at namespace scope",
				Input:      withScope(treetesting.BuildTree(t, withPath(vt.Helper.AdminRoleBinding(), "namespaces/cr.yaml"))),
				ExpectNoop: true,
			},
			{
				Name:      "unknown namespace resource",
				Input:     withScope(treetesting.BuildTree(t, withPath(vt.Helper.UnknownResource(), "namespaces/cr.yaml"))),
				ExpectErr: true,
			},
			{
				Name:      "unknown cluster resource",
				Input:     withScope(treetesting.BuildTree(t, withPath(vt.Helper.UnknownResource(), "cluster/cr.yaml"))),
				ExpectErr: true,
			},
		},
	}
	t.Run("scope", scopeTestcases.Run)
}
