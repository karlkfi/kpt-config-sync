package tree

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/analyzer/vet/vettesting"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
)

type directoryTreeTestcase struct {
	name   string
	inputs []string
	expect *ast.TreeNode
}

func (tc *directoryTreeTestcase) Run(t *testing.T) {
	tg := newDirectoryTree()
	for _, dir := range tc.inputs {
		tg.addDir(cmpath.FromSlash(dir))
	}
	var eb status.MultiError
	tr := tg.build()

	vettesting.ExpectErrors(nil, eb, t)

	if diff := cmp.Diff(tc.expect, tr); diff != "" {
		_, err := spew.Printf("%#v\n", tr)
		t.Error(err)
		t.Fatalf("unexpected output:\n%s", diff)
	}
}

func TestDirectoryTree(t *testing.T) {
	var directoryTreeTestcases = []directoryTreeTestcase{
		{
			name:   "only root",
			inputs: []string{"a"},
			expect: &ast.TreeNode{
				Path: cmpath.FromSlash("a"),
				Type: node.AbstractNamespace,
			},
		},
		{
			name:   "root second",
			inputs: []string{"a/b", "a"},
			expect: &ast.TreeNode{
				Path: cmpath.FromSlash("a"),
				Type: node.AbstractNamespace,
				Children: []*ast.TreeNode{
					{
						Path: cmpath.FromSlash("a/b"),
						Type: node.AbstractNamespace,
					},
				},
			},
		},
		{
			name:   "missing root",
			inputs: []string{"a/b"},
			expect: &ast.TreeNode{
				Path: cmpath.FromSlash("a"),
				Type: node.AbstractNamespace,
				Children: []*ast.TreeNode{
					{
						Path: cmpath.FromSlash("a/b"),
						Type: node.AbstractNamespace,
					},
				},
			},
		},
		{
			name:   "small out of order tree",
			inputs: []string{"a", "a/b/c", "a/b"},
			expect: &ast.TreeNode{
				Path: cmpath.FromSlash("a"),
				Type: node.AbstractNamespace,
				Children: []*ast.TreeNode{
					{
						Path: cmpath.FromSlash("a/b"),
						Type: node.AbstractNamespace,
						Children: []*ast.TreeNode{
							{
								Path: cmpath.FromSlash("a/b/c"),
								Type: node.AbstractNamespace,
							},
						},
					},
				},
			},
		},
		{
			name:   "two children",
			inputs: []string{"a", "a/b", "a/c"},
			expect: &ast.TreeNode{
				Path: cmpath.FromSlash("a"),
				Type: node.AbstractNamespace,
				Children: []*ast.TreeNode{
					{
						Path: cmpath.FromSlash("a/b"),
						Type: node.AbstractNamespace,
					},
					{
						Path: cmpath.FromSlash("a/c"),
						Type: node.AbstractNamespace,
					},
				},
			},
		},
		{
			name:   "missing node",
			inputs: []string{"a", "a/b/c"},
			expect: &ast.TreeNode{
				Path: cmpath.FromSlash("a"),
				Type: node.AbstractNamespace,
				Children: []*ast.TreeNode{
					{
						Path: cmpath.FromSlash("a/b"),
						Type: node.AbstractNamespace,
						Children: []*ast.TreeNode{
							{
								Path: cmpath.FromSlash("a/b/c"),
								Type: node.AbstractNamespace,
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range directoryTreeTestcases {
		t.Run(tc.name, tc.Run)
	}
}
