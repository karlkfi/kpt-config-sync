package tree

import (
	"sort"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
)

// Builder populates the nodes in the hierarchy tree with their corresponding objects.
type Builder struct {
	objects map[cmpath.Relative][]ast.FileObject
}

// NewBuilder initializes an Builder with the set of objects to use to
// populate the config hierarchy tree.
func NewBuilder(objects []ast.FileObject) *Builder {
	v := &Builder{objects: make(map[cmpath.Relative][]ast.FileObject)}

	for _, object := range objects {
		dir := object.Dir()
		v.objects[dir] = append(v.objects[dir], object)
	}

	for dir := range v.objects {
		sort.Slice(v.objects[dir], func(i, j int) bool {
			return lessFileObject(v.objects[dir][i], v.objects[dir][j])
		})
	}
	return v
}

// VisitRoot creates nodes for the config hierarchy.
func (v *Builder) VisitRoot(r *ast.Root) *ast.Root {
	treeBuilder := newDirectoryTree()
	for dir := range v.objects {
		treeBuilder.addDir(dir)
	}
	r.Tree = treeBuilder.build()

	if r.Tree != nil {
		v.VisitTreeNode(r.Tree)
	}
	return r
}

// VisitTreeNode adds all objects which correspond to the TreeNode in the config hierarchy.
func (v *Builder) VisitTreeNode(n *ast.TreeNode) *ast.TreeNode {
	for _, object := range v.objects[n.Relative] {
		if object.GroupVersionKind() == kinds.Namespace() {
			n.Type = node.Namespace
		}
		n.Objects = append(n.Objects, object)
	}
	for _, child := range n.Children {
		v.VisitTreeNode(child)
	}
	return n
}

func lessFileObject(i, j ast.FileObject) bool {
	// Behavior when objects have the same directory, GVK, and name is undefined.
	igvk := i.GroupVersionKind().String()
	jgvk := j.GroupVersionKind().String()
	if igvk != jgvk {
		return igvk < jgvk
	}
	return i.GetName() < j.GetName()
}
