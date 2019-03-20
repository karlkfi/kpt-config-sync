/*
Copyright 2017 The Nomos Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tree

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/node"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet/vettesting"
	"github.com/google/nomos/pkg/policyimporter/filesystem/cmpath"
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
	eb := status.ErrorBuilder{}
	tr := tg.build()

	vettesting.ExpectErrors(nil, eb.Build(), t)

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
