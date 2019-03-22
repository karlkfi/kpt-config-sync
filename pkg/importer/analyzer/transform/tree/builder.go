/*
Copyright 2017 The CSP Config Management Authors.
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
	"sort"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
)

// builder handles constructing an ast.TreeNode tree from directory paths.
type builder struct {
	// root is the root node of the tree
	root *ast.TreeNode
	// namespaceDirs is a map of relative UNIX-style directory path to node
	nodes map[cmpath.Path]*ast.TreeNode
}

// newDirectoryTree returns a new tree generator
func newDirectoryTree() *builder {
	return &builder{nodes: map[cmpath.Path]*ast.TreeNode{}}
}

func newNode(p cmpath.Path) *ast.TreeNode {
	return &ast.TreeNode{
		Path: p,
		Type: node.AbstractNamespace,
	}
}

// addDir adds a node at the the given path.
// p is the cmpath.Relative of the new ast.TreeNode.
// Recursively adds parent nodes as necessary until it reaches the policy hierarchy root.
func (t *builder) addDir(dir cmpath.Path) {
	if t.nodes[dir] != nil {
		return
	}
	curNode := newNode(dir)
	for curDir := dir; ; {
		t.nodes[curDir] = curNode

		parentDir := curDir.Dir()
		if parentDir.IsRoot() {
			t.root = curNode
			// Stop because `curNode` is the top-level policy hierarchy directory.
			break
		}
		parent := t.nodes[parentDir]
		if parent != nil {
			// Add the curNode to its parent.
			parent.Children = append(parent.Children, curNode)
			// Stop because we found an existing parent.
			break
		}

		parent = newNode(parentDir)
		parent.Children = append(parent.Children, curNode)

		curDir = parentDir
		curNode = parent
	}
}

// build takes all the requested node paths and creates a tree, returning the root node.
// Children of nodes are sorted alphabetically by directory path.
func (t *builder) build() *ast.TreeNode {
	for _, n := range t.nodes {
		// Sort the children by their paths to ensure deterministic tree structure.
		sort.Slice(n.Children, func(i, j int) bool {
			return n.Children[i].SlashPath() < n.Children[j].SlashPath()
		})
	}
	return t.root
}
