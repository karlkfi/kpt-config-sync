package ast

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/runtime"
)

// NewFileObject returns an ast.FileObject with the specified underlying
// client.Object and the designated source file.
// TODO(b/179532046): This function should accept an unstructured.Unstructured
// if possible. Also we should see if we can make FileObject *not* implement
// core.Object and instead make callers explicitly interact with one format or
// the other.
func NewFileObject(object core.Object, source cmpath.Relative) FileObject {
	return FileObject{
		Object:   object,
		Relative: source,
	}
}

// FileObject extends core.Object to include the path to the file in the repo.
type FileObject struct {
	// Object is the unstructured representation of the object.
	core.Object
	// Relative is the path of this object in the repo prefixed by the Nomos Root.
	cmpath.Relative
}

var _ id.Resource = &FileObject{}

// CompareFileObject is a cmp.Option which allows tests to compare FileObjects.
var CompareFileObject = cmp.AllowUnexported(FileObject{})

// DeepCopy returns a deep copy of the FileObject.
func (o *FileObject) DeepCopy() FileObject {
	obj := core.DeepCopy(o.Object)
	return FileObject{
		Object:   obj,
		Relative: o.Relative,
	}
}

// Structured returns the structured representation of the object. This can be
// cast to a golang struct (eg v1.CustomResourceDefinition) for validation and
// hydration logic. Note that the structured object should only be read. No
// mutations to the structured object (eg in hydration) will be persisted.
// Unmarshalling and re-marshalling an object can result in spurious JSON fields
// depending on what directives are specified for those  fields. To be safe, we
// keep all resources in their raw unstructured format.  If hydration or
// validation code requires the structured format, we can convert it here
// separate from the raw unstructured representation.
func (o *FileObject) Structured() (runtime.Object, status.Error) {
	obj, err := core.RemarshalToStructured(o.Object)
	if err != nil {
		return nil, core.ObjectParseError(o.Object, err)
	}
	return obj, nil
}
