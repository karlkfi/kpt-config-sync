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
	"path"
	"path/filepath"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/pkg/errors"
)

// DirectoryTree handles constructing an ast.TreeNode tree from directory paths.
type DirectoryTree struct {
	// rootParent is the parent directory of the root node.
	// Uses OS-specific path separators
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

// SetRootDir sets the given OS-specific path as the root path.
// p is the filepath of the root directory.
// typ denotes whether the root directory is a policyspace or a namespace.
func (t *DirectoryTree) SetRootDir(p string, typ ast.TreeNodeType) *ast.TreeNode {
	if t.root != nil {
		panic("programmer error, cannot set root dir multiple times")
	}
	t.rootParent = filepath.Dir(p)
	t.root = &ast.TreeNode{
		Path:      path.Base(filepath.ToSlash(p)),
		Type:      typ,
		Selectors: map[string]*v1alpha1.NamespaceSelector{},
	}
	t.nodes[t.root.Path] = t.root
	return t.root
}

// AddDir adds the given node at the the given OS-specific path.
// p is the filepath of the directory.
// typ denotes whether the directory is a policyspace or a namespace.
func (t *DirectoryTree) AddDir(p string, typ ast.TreeNodeType) *ast.TreeNode {
	if t.root == nil {
		panic("programmer error, cannot add dir without root")
	}
	relPath, err := filepath.Rel(t.rootParent, p)
	if err != nil {
		panic(fmt.Sprintf("programmer error, should not be non-relative path here: %s", err))
	}
	n := &ast.TreeNode{
		Path:      filepath.ToSlash(relPath),
		Type:      typ,
		Selectors: map[string]*v1alpha1.NamespaceSelector{},
	}
	t.nodes[relPath] = n
	return n
}

// Build takes all the created nodes and produces a tree.
func (t *DirectoryTree) Build() (*ast.TreeNode, error) {
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
