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
	"path"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/node"
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	"github.com/google/nomos/pkg/util/multierror"
)

// DirectoryTree handles constructing an ast.TreeNode tree from directory paths.
type DirectoryTree struct {
	// root is the root node of the tree
	root *ast.TreeNode
	// nodes is a map of relative UNIX-style directory path to node
	nodes map[string]*ast.TreeNode
}

// NewDirectoryTree returns a new tree generator
func NewDirectoryTree() *DirectoryTree {
	return &DirectoryTree{nodes: map[string]*ast.TreeNode{}}
}

// AddDir adds the given node at the the given OS-specific path.
// p is the OS-specific filepath of the directory relative to the Nomos repo root directory.
// typ denotes whether the directory is a policyspace or a namespace.
func (t *DirectoryTree) AddDir(p nomospath.Relative, typ node.Type) *ast.TreeNode {
	newNode := &ast.TreeNode{
		Relative:  p,
		Type:      typ,
		Selectors: map[string]*v1alpha1.NamespaceSelector{},
	}
	t.nodes[newNode.RelativeSlashPath()] = newNode

	if t.root == nil {
		t.root = newNode
	}
	return newNode
}

// Build takes all the created nodes and produces a tree.
func (t *DirectoryTree) Build(eb *multierror.Builder) *ast.TreeNode {
	for p, n := range t.nodes {
		parent := path.Dir(p)
		if parent != "." {
			parentNode, ok := t.nodes[parent]
			if !ok {
				eb.Add(veterrors.InternalErrorf("failed to treeify policy nodes: Node %q missing parent %q", p, parent))
				return nil
			}
			parentNode.Children = append(parentNode.Children, n)
		}
	}
	return t.root
}
