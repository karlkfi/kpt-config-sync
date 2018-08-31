// Reviewed by sunilarora
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
	"fmt"
	"path/filepath"

	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/pkg/errors"
)

// DirectoryTree handles constructing an ast.TreeNode tree from directory paths.
type DirectoryTree struct {
	// rootParent is the parent directory of the root node
	rootParent string
	// root is the root node of the tree
	root *ast.TreeNode
	// nodes is a map of relative directory path to node
	nodes map[string]*ast.TreeNode
}

// NewDirectoryTree returns a new tree generator
func NewDirectoryTree() *DirectoryTree {
	return &DirectoryTree{nodes: map[string]*ast.TreeNode{}}
}

// SetRootDir creates a root node and stores the parent path
func (t *DirectoryTree) SetRootDir(path string) *ast.TreeNode {
	if t.root != nil {
		panic("programmer error, cannot set root dir multiple times")
	}
	t.rootParent = filepath.Dir(path)
	t.root = &ast.TreeNode{
		Path: filepath.Base(path),
		Type: ast.Policyspace,
	}
	t.nodes[t.root.Path] = t.root
	return t.root
}

// AddDir creates a node and converts the path to a path relative to the root's parent.
func (t *DirectoryTree) AddDir(path string, typ ast.TreeNodeType) *ast.TreeNode {
	if t.root == nil {
		panic("programmer error, cannot add dir without root")
	}
	relPath, err := filepath.Rel(t.rootParent, path)
	if err != nil {
		panic(fmt.Sprintf("programmer error, should not be non-relative path here: %s", err))
	}
	n := &ast.TreeNode{
		Path: relPath,
		Type: typ,
	}
	t.nodes[relPath] = n
	return n
}

// Build takes all the created nodes and produces a tree.
func (t *DirectoryTree) Build() (*ast.TreeNode, error) {
	if t.root == nil {
		return nil, errors.Errorf("Missing root node")
	}
	for path, node := range t.nodes {
		parent := filepath.Dir(path)
		if parent != "." {
			parentNode, ok := t.nodes[parent]
			if !ok {
				return nil, errors.Errorf("Node %q missing parent %q", path, parent)
			}
			parentNode.Children = append(parentNode.Children, node)
		}
	}
	return t.root, nil
}
