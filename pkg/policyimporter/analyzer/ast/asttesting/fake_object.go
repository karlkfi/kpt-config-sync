package asttesting

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NewFakeFileObject returns a fake FileObject for testing.
//
// gvk is the GroupVersionKind the FileObject will pretend to be.
// path is the desired slash path relative to nomos root.
func NewFakeFileObject(gvk schema.GroupVersionKind, path string) ast.FileObject {
	return ast.FileObject{
		Object:   NewFakeObject(gvk),
		Relative: nomospath.NewRelative(path),
	}
}

// NewFakeObject returns a fake runtime.Object for testing.
// This is for when the only things you care about are the Kind and Metadata.
func NewFakeObject(gvk schema.GroupVersionKind) *FakeObject {
	return &FakeObject{
		ObjectMeta: &v1.ObjectMeta{},
		TypeMeta: &v1.TypeMeta{
			Kind:       gvk.Kind,
			APIVersion: gvk.GroupVersion().String(),
		},
	}
}

// FakeObject implements a bare bones version of runtime.Object for testing.
type FakeObject struct {
	*v1.ObjectMeta
	*v1.TypeMeta
}

var _ runtime.Object = &FakeObject{}

var _ v1.Object = FakeObject{}

// WithMeta returns a copy of the object with the metadata replaced with meta.
func (o *FakeObject) WithMeta(meta *v1.ObjectMeta) *FakeObject {
	return &FakeObject{
		ObjectMeta: meta,
		TypeMeta:   o.TypeMeta,
	}
}

// WithName returns a copy of the object with the metadata.Name set to the desired value.
func (o *FakeObject) WithName(name string) *FakeObject {
	meta := o.ObjectMeta.DeepCopy()
	meta.SetName(name)
	return &FakeObject{
		ObjectMeta: meta,
		TypeMeta:   o.TypeMeta,
	}
}

// DeepCopyObject implements runtime.Object.
func (o *FakeObject) DeepCopyObject() runtime.Object {
	return &FakeObject{
		ObjectMeta: o.ObjectMeta.DeepCopy(),
		TypeMeta:   o.TypeMeta,
	}
}
