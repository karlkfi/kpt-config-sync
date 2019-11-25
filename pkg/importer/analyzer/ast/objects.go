package ast

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/id"
)

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

// FileObject extends runtime.FileObject to include the path to the file in the repo.
type FileObject struct {
	core.Object
	// Path is the path this object has relative to Nomos Root, if known.
	cmpath.Path
}

var _ id.Resource = &FileObject{}

// DeepCopy returns a deep copy of the FileObject.
func (o *FileObject) DeepCopy() FileObject {
	return FileObject{Object: core.DeepCopy(o.Object), Path: o.Path}
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
