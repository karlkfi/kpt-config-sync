package tree

import (
	"sort"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/node"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuilderVisitor populates the nodes in the hierarchy tree with their corresponding objects.
type BuilderVisitor struct {
	*visitor.Base
	objects map[nomospath.Relative][]ast.FileObject

	// dirs holds all directories
	// we have to maintain these as we treat these two cases differently
	//   1. directory exists but has no resources
	//   2. directory does not exist
	dirs []nomospath.Relative
}

// NewBuilderVisitor initializes an BuilderVisitor with the set of objects to use to
// populate the policy hierarchy tree.
func NewBuilderVisitor(objects []ast.FileObject) *BuilderVisitor {
	v := &BuilderVisitor{Base: visitor.NewBase(), objects: make(map[nomospath.Relative][]ast.FileObject)}
	v.SetImpl(v)

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

// WithDirs adds the specified directories to the Builder.
func (v *BuilderVisitor) WithDirs(dirs []nomospath.Relative) *BuilderVisitor {
	v.dirs = dirs
	return v
}

// VisitRoot creates nodes for the policy hierarchy.
func (v *BuilderVisitor) VisitRoot(r *ast.Root) *ast.Root {
	treeBuilder := newDirectoryTree()
	for dir := range v.objects {
		treeBuilder.addDir(dir)
	}

	// Preserves behavior that distinguishes between directories that have no resources or children
	// and directories that do not exist.
	for _, dir := range v.dirs {
		treeBuilder.addDir(dir)
	}

	r.Tree = treeBuilder.build()
	return v.Base.VisitRoot(r)
}

// VisitTreeNode adds all objects which correspond to the TreeNode in the policy hierarchy.
func (v *BuilderVisitor) VisitTreeNode(n *ast.TreeNode) *ast.TreeNode {
	for _, object := range v.objects[n.Relative] {
		switch o := object.Object.(type) {
		case *corev1.Namespace:
			n.Type = node.Namespace
			metaObj := object.Object.(metav1.Object)
			n.Labels = metaObj.GetLabels()
			n.Annotations = metaObj.GetAnnotations()
		case *v1alpha1.NamespaceSelector:
			if n.Selectors == nil {
				n.Selectors = make(map[string]*v1alpha1.NamespaceSelector)
			}
			n.Selectors[o.Name] = o
		}
		n.Objects = append(n.Objects, &ast.NamespaceObject{FileObject: object})
	}
	return v.Base.VisitTreeNode(n)
}

// RequiresValidState marks that the repository should otherwise be in a valid state before
// attempting to construct the policy hierarchy tree.
func (v *BuilderVisitor) RequiresValidState() bool {
	return true
}

func lessFileObject(i, j ast.FileObject) bool {
	// Behavior when objects have the same directory, GVK, and name is undefined.
	igvk := i.GroupVersionKind().String()
	jgvk := j.GroupVersionKind().String()
	if igvk != jgvk {
		return igvk < jgvk
	}
	return i.Name() < j.Name()
}
