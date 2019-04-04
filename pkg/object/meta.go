package object

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// Namespace replaces the metadata.namesapce of the FileObject under test.
func Namespace(namespace string) Mutator {
	return func(o *ast.FileObject) {
		o.MetaObject().SetNamespace(namespace)
	}
}

// Name replaces the metadata.name of the FileObject under test.
func Name(name string) Mutator {
	return func(o *ast.FileObject) {
		o.MetaObject().SetName(name)
	}
}

// Label adds label=value to the metadata.labels of the FileObject under test.
func Label(label, value string) Mutator {
	return func(o *ast.FileObject) {
		SetLabel(o.MetaObject(), label, value)
	}
}

// Labels sets the object's labels to a copy of the passed map.
// Setting to nil causes a call to SetLabels(nil), but the underlying implementation may set Labels
// to empty map.
func Labels(labels map[string]string) Mutator {
	return func(o *ast.FileObject) {
		SetLabels(o.MetaObject(), labels)
	}
}

// Annotation adds annotation=value to the metadata.annotations of the FileObject under test.
func Annotation(annotation, value string) Mutator {
	return func(o *ast.FileObject) {
		SetAnnotation(o.MetaObject(), annotation, value)
	}
}

// Annotations sets the object's annotations to a copy of the passed map.
// Setting to nil causes a call to SetAnnotations(nil), but the underlying implementation may set
// Annotations to empty map.
func Annotations(annotations map[string]string) Mutator {
	return func(o *ast.FileObject) {
		SetAnnotations(o.MetaObject(), annotations)
	}
}

// OwnerReference sets the object's ownerReference.
func OwnerReference(name, uid string, gvk schema.GroupVersionKind) Mutator {
	return func(o *ast.FileObject) {
		apiVersion, kind := gvk.ToAPIVersionAndKind()
		o.MetaObject().SetOwnerReferences([]metav1.OwnerReference{{
			Name:       name,
			UID:        types.UID(uid),
			APIVersion: apiVersion,
			Kind:       kind,
		}})
	}
}
