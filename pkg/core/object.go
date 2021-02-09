package core

import (
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// LabeledAndAnnotated is a convenience interface.
// Labels and annotations are optional per the Kubernetes API spec, we require
// them as a guaranteed place to store bookkeeping metadata.
//
// This is a subset of the interface metav1.Object, and allows us to manipulate
// AST objects with the same code that operates on Kubernetes API objects,
// without the need to implement parts of metav1.Object that don't deal with
// labels and annotations.
type LabeledAndAnnotated interface {
	GetAnnotations() map[string]string
	SetAnnotations(map[string]string)

	GetLabels() map[string]string
	SetLabels(map[string]string)
}

// Object defines the minimal interface we expect any resource we allow Nomos to sync.
type Object interface {
	client.Object

	// GroupVersionKind overlaps with the client.Object declaration, but avoids having to call
	// o.GetObjectKind().GroupVersionKind() everywhere.
	GroupVersionKind() schema.GroupVersionKind
}

// DeepCopy returns Object rather than client.Object after deep copying.
// We can't define this directly on Object as interfaces may not define methods.
func DeepCopy(o Object) Object {
	// This unchecked cast is safe as DeepCopyObject returns an object of the same type.
	return o.DeepCopyObject().(Object)
}

// OwnerReferenced matches types whose metadata type define ownerReferences
type OwnerReferenced interface {
	GetOwnerReferences() []metav1.OwnerReference
	SetOwnerReferences([]metav1.OwnerReference)
}

// ObjectOf safely attempts to coerce a runtime.Object into a core.Object.
//
// We expect this will never return error in production.
func ObjectOf(o runtime.Object) (Object, error) {
	co, ok := o.(Object)
	if !ok {
		return nil, errors.Errorf("not a Kubernetes object %v", o)
	}
	return co, nil
}
