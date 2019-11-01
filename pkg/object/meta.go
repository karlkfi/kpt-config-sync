package object

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// MetaMutator is a Mutator that modifies the metadata of an Object.
type MetaMutator func(meta metav1.Object)

// Mutate implements object.Mutator.
func (m MetaMutator) Mutate(object *ast.FileObject) {
	m(object.MetaObject())
}

// Namespace replaces the metadata.namespace of the MetaObjectunder test.
func Namespace(namespace string) MetaMutator {
	return func(o metav1.Object) {
		o.SetNamespace(namespace)
	}
}

// Name replaces the metadata.name of the MetaObjectunder test.
func Name(name string) MetaMutator {
	return func(o metav1.Object) {
		o.SetName(name)
	}
}

// Label adds label=value to the metadata.labels of the MetaObjectunder test.
func Label(label, value string) MetaMutator {
	return func(o metav1.Object) {
		SetLabel(o, label, value)
	}
}

// Labels sets the object's labels to a copy of the passed map.
// Setting to nil causes a call to SetLabels(nil), but the underlying implementation may set Labels
// to empty map.
func Labels(labels map[string]string) MetaMutator {
	return func(o metav1.Object) {
		SetLabels(o, labels)
	}
}

// Annotation adds annotation=value to the metadata.annotations of the MetaObject under test.
func Annotation(annotation, value string) MetaMutator {
	return func(o metav1.Object) {
		SetAnnotation(o, annotation, value)
	}
}

// WithoutAnnotation removes annotation from metadata.annotations of the MetaObject under test.
func WithoutAnnotation(annotation string) MetaMutator {
	return func(o metav1.Object) {
		RemoveAnnotations(o, annotation)
	}
}

// Annotations sets the object's annotations to a copy of the passed map.
// Setting to nil causes a call to SetAnnotations(nil), but the underlying implementation may set
// Annotations to empty map.
func Annotations(annotations map[string]string) MetaMutator {
	return func(o metav1.Object) {
		SetAnnotations(o, annotations)
	}
}

// OwnerReference sets the object's ownerReference.
func OwnerReference(name, uid string, gvk schema.GroupVersionKind) MetaMutator {
	return func(o metav1.Object) {
		apiVersion, kind := gvk.ToAPIVersionAndKind()
		o.SetOwnerReferences([]metav1.OwnerReference{{
			Name:       name,
			UID:        types.UID(uid),
			APIVersion: apiVersion,
			Kind:       kind,
		}})
	}
}

// MetaMutators merges multiple MetaMutators into one.
func MetaMutators(mutators ...MetaMutator) MetaMutator {
	return func(o metav1.Object) {
		for _, mutator := range mutators {
			mutator(o)
		}
	}
}
