package core

import (
	"k8s.io/apimachinery/pkg/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MetaMutator is a Mutator that modifies the metadata of an Object.
type MetaMutator func(o Object)

// Namespace replaces the metadata.namespace of the Object under test.
func Namespace(namespace string) MetaMutator {
	return func(o Object) {
		o.SetNamespace(namespace)
	}
}

// Name replaces the metadata.name of the Object under test.
func Name(name string) MetaMutator {
	return func(o Object) {
		o.SetName(name)
	}
}

// UID replaces the metadata.uid of the Object under test.
func UID(uid types.UID) MetaMutator {
	return func(o Object) {
		o.SetUID(uid)
	}
}

// Label adds label=value to the metadata.labels of the Object under test.
func Label(label, value string) MetaMutator {
	return func(o Object) {
		SetLabel(o, label, value)
	}
}

// Labels sets the object's labels to a copy of the passed map.
// Setting to nil causes a call to SetLabels(nil), but the underlying implementation may set Labels
// to empty map.
func Labels(labels map[string]string) MetaMutator {
	return func(o Object) {
		o.SetLabels(copyMap(labels))
	}
}

// Annotation adds annotation=value to the metadata.annotations of the MetaObject under test.
func Annotation(annotation, value string) MetaMutator {
	return func(o Object) {
		SetAnnotation(o, annotation, value)
	}
}

// WithoutAnnotation removes annotation from metadata.annotations of the MetaObject under test.
func WithoutAnnotation(annotation string) MetaMutator {
	return func(o Object) {
		RemoveAnnotations(o, annotation)
	}
}

// Annotations sets the object's annotations to a copy of the passed map.
// Setting to nil causes a call to SetAnnotations(nil), but the underlying implementation may set
// Annotations to empty map.
func Annotations(annotations map[string]string) MetaMutator {
	return func(o Object) {
		o.SetAnnotations(copyMap(annotations))
	}
}

// OwnerReference sets the object's owner references to a passed slice of metav1.OwnerReference.
func OwnerReference(or []metav1.OwnerReference) MetaMutator {
	return func(o Object) {
		o.SetOwnerReferences(or)
	}
}

// Generation replaces the metadata.generation of the Object under test.
func Generation(gen int64) MetaMutator {
	return func(o Object) {
		o.SetGeneration(gen)
	}
}
