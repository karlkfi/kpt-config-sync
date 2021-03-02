package validate

import (
	"errors"
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/validate/objects"
)

func TestNamespaceSelector(t *testing.T) {
	testCases := []struct {
		name     string
		objs     *objects.Tree
		wantErrs status.MultiError
	}{
		{
			name: "NamespaceSelector in abstract namespace",
			objs: &objects.Tree{
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Objects: []*ast.NamespaceObject{
						{FileObject: fake.NamespaceSelector()},
					},
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/hello"),
							Type:     node.Namespace,
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.Namespace("namespaces/hello")},
							},
						},
					},
				},
			},
		},
		{
			name: "NamespaceSelector in Namespace",
			objs: &objects.Tree{
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/hello"),
							Type:     node.Namespace,
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.Namespace("namespaces/hello")},
								{FileObject: fake.NamespaceSelector()},
							},
						},
					},
				},
			},
			wantErrs: syntax.IllegalKindInNamespacesError(fake.NamespaceSelector()),
		},
		{
			name: "Role in Namespace",
			objs: &objects.Tree{
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/hello"),
							Type:     node.Namespace,
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.Namespace("namespaces/hello")},
								{FileObject: fake.RoleAtPath("namespaces/hello/role.yaml")},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errs := NamespaceSelector(tc.objs)
			if !errors.Is(errs, tc.wantErrs) {
				t.Errorf("got NamespaceSelector() error %v, want %v", errs, tc.wantErrs)
			}
		})
	}
}
