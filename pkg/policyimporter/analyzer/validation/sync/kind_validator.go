package sync

import (
	"github.com/google/nomos/pkg/api/policyhierarchy"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/scale/scheme/extensionsv1beta1"
)

// KindValidatorFactory ensures that only supported Resource Kinds are declared in Syncs.
var KindValidatorFactory = ValidatorFactory{fn: func(sync FileGroupVersionKindHierarchySync) error {
	if allowedInSyncs(sync.GroupVersionKind) {
		return nil
	}
	return vet.UnsupportedResourceInSyncError{
		Source: sync.Source,
		GVK:    sync.GroupVersionKind,
	}
}}

// allowedInSyncs returns true if the passed GVK is allowed to be declared in Syncs.
func allowedInSyncs(gvk schema.GroupVersionKind) bool {
	return !unsupportedSyncResources()[gvk] && (gvk.Group != policyhierarchy.GroupName)
}

// unsupportedSyncResources returns a map of each type where syncing is explicitly not supported.
func unsupportedSyncResources() map[schema.GroupVersionKind]bool {
	return map[schema.GroupVersionKind]bool{
		customResourceDefinition(): true,
		namespace():                true,
	}
}

// customResourceDefinition is temporary. It is moved and documented in the already-approved next CL.
func customResourceDefinition() schema.GroupVersionKind {
	return extensionsv1beta1.SchemeGroupVersion.WithKind("CustomResourceDefinition")
}

// namespace is temporary. It is moved and documented in the already-approved next CL.
func namespace() schema.GroupVersionKind {
	return corev1.SchemeGroupVersion.WithKind("Namespace")
}
