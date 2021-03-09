// Package hnc adds additional HNC-understandable annotation and labels to namespaces managed by
// ACM. Please send code reviews to gke-kubernetes-hnc-core@.
package hnc

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/resourcequota"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestBuilderVisitor(t *testing.T) {
	testCases := []struct {
		name         string
		input        *ast.Root
		expectOutput *ast.Root
	}{
		{
			name: "label and annotate namespace",
			input: &ast.Root{
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
									Objects:  []*ast.NamespaceObject{{FileObject: fake.Namespace("namespaces/foo/bar")}},
								},
							},
						},
						{
							Relative: cmpath.RelativeSlash("namespaces/qux"),
							Type:     node.Namespace,
							Objects:  []*ast.NamespaceObject{{FileObject: fake.Namespace("namespaces/qux")}},
						},
					},
				},
			},
			expectOutput: &ast.Root{
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
									Objects: []*ast.NamespaceObject{{FileObject: fake.Namespace(
										"namespaces/foo/bar",
										core.Annotation(AnnotationKeyV1A2, v1.ManagedByValue),
										core.Label("foo.tree.hnc.x-k8s.io/depth", "1"),
										core.Label("bar.tree.hnc.x-k8s.io/depth", "0"))}},
								},
							},
						},
						{
							Relative: cmpath.RelativeSlash("namespaces/qux"),
							Type:     node.Namespace,
							Objects: []*ast.NamespaceObject{{FileObject: fake.Namespace("namespaces/qux",
								core.Annotation(AnnotationKeyV1A2, v1.ManagedByValue),
								core.Label("qux.tree.hnc.x-k8s.io/depth", "0"))}},
						},
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := []cmp.Option{resourcequota.ResourceQuantityEqual(), cmp.AllowUnexported(), ast.CompareFileObject}

			copier := visitor.NewBase()
			copier.SetImpl(copier)
			inputCopy := tc.input.Accept(copier)

			v := NewNamespaceVisitor()
			actual := tc.input.Accept(v)
			if !cmp.Equal(tc.input, inputCopy, opts...) {
				t.Errorf("Input mutated while running visitor: %s", cmp.Diff(inputCopy, tc.input, opts...))
			}

			if !cmp.Equal(tc.expectOutput, actual, opts...) {
				t.Errorf("mismatch on expected vs actual:\ndiff:\n%s",
					cmp.Diff(tc.expectOutput, actual, opts...))
			}

		})
	}
}
