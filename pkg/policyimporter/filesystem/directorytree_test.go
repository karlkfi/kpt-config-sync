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

package filesystem

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/node"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	"github.com/google/nomos/pkg/util/multierror"
)

type directoryTreeInput struct {
	path string
	typ  node.Type
}

type directoryTreeTestcase struct {
	name      string
	inputs    []directoryTreeInput
	expect    *ast.TreeNode
	expectErr bool
}

func (tc *directoryTreeTestcase) Run(t *testing.T) {
	tg := NewDirectoryTree()
	for _, inp := range tc.inputs {
		typ := inp.typ
		if typ == "" {
			typ = node.AbstractNamespace
		}
		n := tg.AddDir(nomospath.NewFakeRelative(inp.path), typ)
		if n == nil {
			t.Errorf("AddNode returned nil")
		}
	}
	eb := multierror.Builder{}
	tree := tg.Build(&eb)
	if eb.HasErrors() != tc.expectErr {
		if tc.expectErr {
			t.Errorf("Expected err, got nil")
		} else {
			t.Errorf("Unexpected error %v", eb.Build())
		}
	}

	if diff := cmp.Diff(tc.expect, tree, ast.FileObjectCmp()); diff != "" {
		spew.Printf("%#v\n", tree)
		t.Errorf("unexpected output:\n%s", diff)
	}
}

var directoryTreeTestcases = []directoryTreeTestcase{
	{
		name: "only root",
		inputs: []directoryTreeInput{
			{path: "a"},
		},
		expect: &ast.TreeNode{
			Relative:  nomospath.NewFakeRelative("a"),
			Type:      node.AbstractNamespace,
			Selectors: map[string]*v1alpha1.NamespaceSelector{},
		},
	},
	{
		name: "small tree",
		inputs: []directoryTreeInput{
			{path: "a"},
			{path: "a/b/c", typ: node.Namespace},
			{path: "a/b"},
		},
		expect: &ast.TreeNode{
			Relative:  nomospath.NewFakeRelative("a"),
			Type:      node.AbstractNamespace,
			Selectors: map[string]*v1alpha1.NamespaceSelector{},
			Children: []*ast.TreeNode{
				{
					Relative:  nomospath.NewFakeRelative("a/b"),
					Type:      node.AbstractNamespace,
					Selectors: map[string]*v1alpha1.NamespaceSelector{},
					Children: []*ast.TreeNode{
						{
							Relative:  nomospath.NewFakeRelative("a/b/c"),
							Type:      node.Namespace,
							Selectors: map[string]*v1alpha1.NamespaceSelector{},
						},
					},
				},
			},
		},
	},
	{
		name: "missing node",
		inputs: []directoryTreeInput{
			{path: "/a/b/c"},
			{path: "/a/b/c/d/e", typ: node.Namespace},
		},
		expectErr: true,
	},
}

func TestDirectoryTree(t *testing.T) {
	for _, tc := range directoryTreeTestcases {
		t.Run(tc.name, tc.Run)
	}
}
