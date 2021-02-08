package hnc

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	oldhnc "github.com/google/nomos/pkg/importer/analyzer/hnc"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/validate/parsed"
)

func TestBuilderVisitor(t *testing.T) {
	testCases := []struct {
		name string
		root *parsed.TreeRoot
		want *parsed.TreeRoot
	}{
		{
			name: "label and annotate namespace",
			root: &parsed.TreeRoot{
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/foo"),
							Type:     node.AbstractNamespace,
							Children: []*ast.TreeNode{
								{
									Relative: cmpath.RelativeSlash("namespaces/foo/bar"),
									Type:     node.Namespace,
									Objects: []*ast.NamespaceObject{
										{FileObject: fake.Namespace("namespaces/foo/bar")},
									},
								},
							},
						},
						{
							Relative: cmpath.RelativeSlash("namespaces/qux"),
							Type:     node.Namespace,
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.Namespace("namespaces/qux")},
							},
						},
					},
				},
			},
			want: &parsed.TreeRoot{
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/foo"),
							Type:     node.AbstractNamespace,
							Children: []*ast.TreeNode{
								{
									Relative: cmpath.RelativeSlash("namespaces/foo/bar"),
									Type:     node.Namespace,
									Objects: []*ast.NamespaceObject{
										{FileObject: fake.Namespace("namespaces/foo/bar",
											core.Annotation(oldhnc.AnnotationKeyV1A1, v1.ManagedByValue),
											core.Annotation(oldhnc.AnnotationKeyV1A2, v1.ManagedByValue),
											core.Label("foo.tree.hnc.x-k8s.io/depth", "1"),
											core.Label("bar.tree.hnc.x-k8s.io/depth", "0"))},
									},
								},
							},
						},
						{
							Relative: cmpath.RelativeSlash("namespaces/qux"),
							Type:     node.Namespace,
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.Namespace("namespaces/qux",
									core.Annotation(oldhnc.AnnotationKeyV1A1, v1.ManagedByValue),
									core.Annotation(oldhnc.AnnotationKeyV1A2, v1.ManagedByValue),
									core.Label("qux.tree.hnc.x-k8s.io/depth", "0"))},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dh := DepthHydrator()
			if err := dh.Hydrate(tc.root); err != nil {
				t.Errorf("Got Hydrate() error %v, want nil", err)
			}
			if diff := cmp.Diff(tc.want, tc.root, ast.CompareFileObject); diff != "" {
				t.Error(diff)
			}
		})
	}
}
