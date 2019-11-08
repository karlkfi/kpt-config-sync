package ast

import (
	"time"

	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/id"
)

// FileObject extends runtime.FileObject to include the path to the file in the repo.
type FileObject struct {
	core.Object
	// Path is the path this object has relative to Nomos Root, if known.
	cmpath.Path
}

var _ id.Resource = &FileObject{}

// NewFileObject returns an ast.FileObject with the specified underlying runtime.Object and the
// designated source file.
func NewFileObject(object core.Object, source cmpath.Path) FileObject {
	return FileObject{Object: object, Path: source}
}

// ParseFileObject returns a FileObject initialized from the given runtime.Object and a valid source
// path parsed from its annotations.
func ParseFileObject(o core.Object) *FileObject {
	srcPath := cmpath.FromSlash(o.GetAnnotations()[v1.SourcePathAnnotationKey])
	return &FileObject{Object: o, Path: srcPath}
}

// DeepCopy returns a deep copy of the FileObject.
func (o *FileObject) DeepCopy() FileObject {
	return FileObject{Object: core.DeepCopy(o.Object), Path: o.Path}
}

// Root represents a hierarchy of declared configs, settings for how those configs will be
// interpreted, and information regarding where those configs came from.
type Root struct {
	// ImportToken is the token for context
	ImportToken string
	LoadTime    time.Time // Time at which the context was generated

	// ClusterName is the name of the Cluster to generate the policy hierarchy for. Determines which
	// ClusterSelectors are active.
	ClusterName string
	Repo        *v1.Repo // Nomos repo

	// ClusterObjects represents resources that are cluster scoped.
	ClusterObjects []*ClusterObject

	// ClusterRegistryObjects represents resources that are related to multi-cluster.
	ClusterRegistryObjects []*ClusterRegistryObject

	// SystemObjects represents resources regarding nomos configuration.
	SystemObjects []*SystemObject

	// Tree represents the directory hierarchy containing namespace scoped resources.
	Tree *TreeNode
	Data *Extension
}

// Accept invokes VisitRoot on the visitor.
func (c *Root) Accept(visitor Visitor) *Root {
	if c == nil {
		return nil
	}
	return visitor.VisitRoot(c)
}

// SystemObject extends FileObject to implement Visitable for cluster scoped objects.
//
// A SystemObject represents a cluster scoped resource from the cluster directory.
type SystemObject struct {
	FileObject
}

// Accept invokes VisitSystemObject on the visitor.
func (o *SystemObject) Accept(visitor Visitor) *SystemObject {
	if o == nil {
		return nil
	}
	return visitor.VisitSystemObject(o)
}

// ClusterRegistryObject extends FileObject to implement Visitable for cluster scoped objects.
//
// A ClusterRegistryObject represents a cluster scoped resource from the cluster directory.
type ClusterRegistryObject struct {
	FileObject
}

// Accept invokes VisitClusterRegistryObject on the visitor.
func (o *ClusterRegistryObject) Accept(visitor Visitor) *ClusterRegistryObject {
	if o == nil {
		return nil
	}
	return visitor.VisitClusterRegistryObject(o)
}

// ClusterObject extends FileObject to implement Visitable for cluster scoped objects.
//
// A ClusterObject represents a cluster scoped resource from the cluster directory.
type ClusterObject struct {
	FileObject
}

// Accept invokes VisitClusterObject on the visitor.
func (o *ClusterObject) Accept(visitor Visitor) *ClusterObject {
	if o == nil {
		return nil
	}
	return visitor.VisitClusterObject(o)
}

// TreeNode is analogous to a directory in the config hierarchy.
type TreeNode struct {
	// Path is the path this node has relative to a nomos Root.
	cmpath.Path

	// The type of the HierarchyNode
	Type        node.Type
	Labels      map[string]string
	Annotations map[string]string

	// Objects from the directory
	Objects []*NamespaceObject

	// Selectors is a map of name to NamespaceSelector objects found at this node.
	// One or more Objects may have an annotation referring to these NamespaceSelectors by name.
	Selectors map[string]*v1.NamespaceSelector

	// Extension holds visitor specific data.
	Data *Extension

	// children of the directory
	Children []*TreeNode
}

var _ id.TreeNode = &TreeNode{}

// Accept invokes VisitTreeNode on the visitor.
func (n *TreeNode) Accept(visitor Visitor) *TreeNode {
	if n == nil {
		return nil
	}
	return visitor.VisitTreeNode(n)
}

// PartialCopy makes an almost shallow copy of n.  An "almost shallow" copy of
// TreeNode make shallow copies of Children and members that are likely
// immutable.  A  deep copy is made of mutable members like Labels and
// Annotations.
func (n *TreeNode) PartialCopy() *TreeNode {
	nn := *n
	copyMapInto(n.Annotations, &nn.Annotations)
	copyMapInto(n.Labels, &nn.Labels)
	// Not sure if Selectors should be copied the same way.
	return &nn
}

// Name returns the name of the lowest-level directory in this node's path.
func (n *TreeNode) Name() string {
	return n.Base()
}

func copyMapInto(from map[string]string, to *map[string]string) {
	if from == nil {
		return
	}
	*to = make(map[string]string)
	for k, v := range from {
		(*to)[k] = v
	}
}

// GetAnnotations returns the annotations from n.  They are mutable if not nil.
func (n *TreeNode) GetAnnotations() map[string]string {
	return n.Annotations
}

// SetAnnotations replaces the annotations on the tree node with the supplied ones.
func (n *TreeNode) SetAnnotations(a map[string]string) {
	n.Annotations = a
}

// NamespaceObject extends FileObject to implement Visitable for namespace scoped objects.
//
// An NamespaceObject represents a resource found in a directory in the config hierarchy.
type NamespaceObject struct {
	FileObject
}

// Accept invokes VisitObject on the visitor.
func (o *NamespaceObject) Accept(visitor Visitor) *NamespaceObject {
	if o == nil {
		return nil
	}
	return visitor.VisitObject(o)
}
