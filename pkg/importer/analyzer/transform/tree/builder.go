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
	// nodes is a map of relative relatives paths from repository root to each TreeNode.
	nodes map[cmpath.Relative]*ast.TreeNode
}

// newDirectoryTree returns a new tree generator
func newDirectoryTree() *builder {
	return &builder{nodes: map[cmpath.Relative]*ast.TreeNode{}}
}

func newNode(p cmpath.Relative) *ast.TreeNode {
	return &ast.TreeNode{
		Relative: p,
		Type:     node.AbstractNamespace,
	}
}

// addDir adds a node at the the given path.
// dir is the path of the new ast.TreeNode relative to repository root.
// Recursively adds parent nodes as necessary until it reaches the config hierarchy root.
func (t *builder) addDir(dir cmpath.Relative) {
	if t.nodes[dir] != nil {
		return
	}
	curNode := newNode(dir)
	for curDir := dir; ; {
		t.nodes[curDir] = curNode

		parentDir := curDir.Dir()
		if parentDir.IsRoot() {
			t.root = curNode
			// Stop because `curNode` is the top-level config hierarchy directory.
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
