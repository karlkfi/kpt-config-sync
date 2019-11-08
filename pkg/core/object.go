package core

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// Named defines the metadata.name field, and is required by the Kubernetes API conventions.
type Named interface {
	GetName() string
	SetName(name string)
}

// Namespaced defines the metadata.namespace field, and is required by the Kubernetes API conventions.
type Namespaced interface {
	GetNamespace() string
	SetNamespace(namespace string)
}

// LabeledAndAnnotated is a convenience interface.
// Labels and annotations are optional per the Kubernetes API spec, we require them as a
// guaranteed place to store bookkeeping metadata.
type LabeledAndAnnotated interface {
	Labeled
	Annotated
}

// resourceVersion is optional per the Kubernetes API spec, but we require it to determine whether
// to update objects. We disallow it for object declarations.
type resourceVersioned interface {
	GetResourceVersion() string
	SetResourceVersion(version string)
}

// Object defines the minimal interface we expect any resource we allow Nomos to sync.
type Object interface {
	Named
	Namespaced
	LabeledAndAnnotated
	resourceVersioned

	// GetUID and SetUID define metadata.uid, which all persistent Kubernetes types must define.
	// Users MUST leave the uid field empty as it is managed by Kubernetes.
	GetUID() types.UID
	SetUID(types.UID)

	// GroupVersionKind overlaps with the runtime.Object declaration, but avoids having to call
	// o.GetObjectKind().GroupVersionKind() everywhere.
	GroupVersionKind() schema.GroupVersionKind

	// Object is Kubernetes's hacky way around Go's lack of generic interfaces, and required for
	// interacting with the Kubernetes APIs.
	runtime.Object
}

// DeepCopy returns Object rather than runtime.Object after deep copying.
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
