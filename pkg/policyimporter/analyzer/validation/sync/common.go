package sync

import (
	"github.com/google/nomos/pkg/api/policyhierarchy"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/scale/scheme/extensionsv1beta1"
)

// isUnsupported returns true if the passed GVK is explicitly not allowed in Syncs.
func isUnsupported(gvk schema.GroupVersionKind) bool {
	return unsupportedSyncResources[gvk] || (gvk.Group == policyhierarchy.GroupName)
}

// unsupportedSyncResources is a set of GroupVersionKinds that are not allowed in Syncs.
var unsupportedSyncResources = map[schema.GroupVersionKind]bool{
	extensionsv1beta1.SchemeGroupVersion.WithKind("CustomResourceDefinition"): true,
	corev1.SchemeGroupVersion.WithKind("Namespace"):                           true,
}

// This is a convenience method which allows iterating over every Kind declaration in
// all Sync Resources in a repository. This avoids having to re-declare loops nested four layers
// deep everywhere.
func kindSyncs(objects []ast.FileObject) []kindSync {
	var result []kindSync
	for _, object := range objects {
		switch o := object.Object.(type) {
		case *v1alpha1.Sync:
			for _, group := range o.Spec.Groups {
				for _, kind := range group.Kinds {
					for _, version := range kind.Versions {
						gvk := schema.GroupVersionKind{
							Group:   group.Group,
							Version: version.Version,
							Kind:    kind.Kind,
						}
						hierarchy := kind.HierarchyMode
						result = append(result, kindSync{sync: vet.ToResourceAddr(object), gvk: gvk, hierarchy: hierarchy})
					}
				}
			}
		}
	}
	return result
}
