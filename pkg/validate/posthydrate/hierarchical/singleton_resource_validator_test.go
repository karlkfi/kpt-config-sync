package hierarchical

import (
	"errors"
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/validate/parsed"
)

func TestSingletonValidator(t *testing.T) {
	testCases := []struct {
		name    string
		root    parsed.Root
		wantErr status.MultiError
	}{
		{
			name: "no error for zero Repos",
			root: &parsed.TreeRoot{
				ClusterObjects: []ast.FileObject{
					fake.ClusterRole(core.Name("reader")),
					fake.ClusterRole(core.Name("writer")),
				},
			},
		},
		{
			name: "no error for one Repo",
			root: &parsed.TreeRoot{
				SystemObjects: []ast.FileObject{
					fake.Repo(),
				},
			},
		},
		{
			name: "error for two Repos",
			root: &parsed.TreeRoot{
				SystemObjects: []ast.FileObject{
					fake.Repo(),
					fake.Repo(core.Name("oops")),
				},
			},
			wantErr: status.MultipleSingletonsError(fake.Repo(), fake.Repo(core.Name("oops"))),
		},
	}

	for _, tc := range testCases {
		sv := SingletonValidator(kinds.Repo())
		t.Run(tc.name, func(t *testing.T) {
			err := sv(tc.root)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("Got SingletonValidator() error %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestTreeNodeSingletonValidator(t *testing.T) {
	testCases := []struct {
		name    string
		root    parsed.Root
		wantErr status.MultiError
	}{
		{
			name: "zero namespaces",
			root: &parsed.TreeRoot{},
		},
		{
			name: "one namespace",
			root: &parsed.TreeRoot{
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
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
			name: "two namespaces in different directories",
			root: &parsed.TreeRoot{
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/hello"),
							Type:     node.Namespace,
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.Namespace("namespaces/hello")},
							},
						},
						{
							Relative: cmpath.RelativeSlash("namespaces/world"),
							Type:     node.Namespace,
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.Namespace("namespaces/world")},
							},
						},
					},
				},
			},
		},
		{
			name: "two namespaces in same directory",
			root: &parsed.TreeRoot{
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/hello"),
							Type:     node.Namespace,
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.Namespace("namespaces/hello")},
								{FileObject: fake.Namespace("namespaces/world")},
							},
						},
					},
				},
			},
			wantErr: status.MultipleSingletonsError(fake.Namespace("namespaces/hello"), fake.Namespace("namespaces/world")),
		},
	}

	for _, tc := range testCases {
		sv := TreeNodeSingletonValidator(kinds.Namespace())
		t.Run(tc.name, func(t *testing.T) {

			err := sv(tc.root)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("Got TreeNodeSingletonValidator() error %v, want %v", err, tc.wantErr)
			}
		})
	}
}
