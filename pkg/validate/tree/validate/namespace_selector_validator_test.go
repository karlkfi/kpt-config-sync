package validate

import (
	"errors"
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/importer/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/metadata"
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
				NamespaceSelectors: map[string]ast.FileObject{
					"dev": fake.NamespaceSelectorAtPath("namespaces/sel.yaml",
						core.Name("dev")),
				},
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/hello"),
							Type:     node.Namespace,
							Objects: []ast.FileObject{
								fake.Namespace("namespaces/hello"),
							},
						},
					},
				},
			},
		},
		{
			name: "NamespaceSelector in Namespace",
			objs: &objects.Tree{
				NamespaceSelectors: map[string]ast.FileObject{
					"dev": fake.NamespaceSelectorAtPath("namespaces/hello/sel.yaml",
						core.Name("dev")),
				},
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/hello"),
							Type:     node.Namespace,
							Objects: []ast.FileObject{
								fake.Namespace("namespaces/hello"),
							},
						},
					},
				},
			},
			wantErrs: fake.Errors(syntax.IllegalKindInNamespacesErrorCode),
		},
		{
			name: "Object references ancestor NamespaceSelector",
			objs: &objects.Tree{
				NamespaceSelectors: map[string]ast.FileObject{
					"dev": fake.NamespaceSelectorAtPath("namespaces/hello/sel.yaml",
						core.Name("dev")),
				},
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/hello"),
							Type:     node.AbstractNamespace,
							Children: []*ast.TreeNode{
								{
									Relative: cmpath.RelativeSlash("namespaces/hello/world"),
									Type:     node.Namespace,
									Objects: []ast.FileObject{
										fake.Namespace("namespaces/hello/world"),
										fake.RoleAtPath("namespaces/hello/world/role.yaml",
											core.Annotation(metadata.NamespaceSelectorAnnotationKey, "dev")),
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Object references non-ancestor NamespaceSelector",
			objs: &objects.Tree{
				NamespaceSelectors: map[string]ast.FileObject{
					"dev": fake.NamespaceSelectorAtPath("namespaces/goodbye/sel.yaml",
						core.Name("dev")),
				},
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/hello"),
							Type:     node.AbstractNamespace,
							Children: []*ast.TreeNode{
								{
									Relative: cmpath.RelativeSlash("namespaces/hello/world"),
									Type:     node.Namespace,
									Objects: []ast.FileObject{
										fake.Namespace("namespaces/hello/world"),
										fake.RoleAtPath("namespaces/hello/world/role.yaml",
											core.Annotation(metadata.NamespaceSelectorAnnotationKey, "dev")),
									},
								},
							},
						},
						{
							Relative: cmpath.RelativeSlash("namespaces/goodbye"),
							Type:     node.AbstractNamespace,
							Children: []*ast.TreeNode{
								{
									Relative: cmpath.RelativeSlash("namespaces/goodbye/moon"),
									Type:     node.Namespace,
									Objects: []ast.FileObject{
										fake.Namespace("namespaces/goodbye/moon"),
										fake.RoleAtPath("namespaces/goodbye/moon/role.yaml"),
									},
								},
							},
						},
					},
				},
			},
			wantErrs: fake.Errors(selectors.ObjectHasUnknownSelectorCode),
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
