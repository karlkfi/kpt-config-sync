package differ

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/hnc"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ManagementEnabled returns true if the resource explicitly has management enabled on a resource
// on the API server.
func ManagementEnabled(obj client.Object) bool {
	return obj.GetAnnotations()[v1.ResourceManagementKey] == v1.ResourceManagementEnabled
}

// ManagementDisabled returns true if the resource in the repo explicitly has management disabled.
func ManagementDisabled(obj client.Object) bool {
	return obj.GetAnnotations()[v1.ResourceManagementKey] == v1.ResourceManagementDisabled
}

// ManagementUnset returns true if the resource has no Nomos ResourceManagementKey.
func ManagementUnset(obj client.Object) bool {
	_, found := obj.GetAnnotations()[v1.ResourceManagementKey]
	return !found
}

// HasNomosMeta returns true if the given map has at least one Nomos annotation or label that syncer
// manages.
func HasNomosMeta(obj client.Object) bool {
	as := obj.GetAnnotations()
	sas := append(v1.SyncerAnnotations(), hnc.AnnotationKeyV1A2)
	for _, a := range sas {
		if _, ok := as[a]; ok {
			return true
		}
	}

	ls := obj.GetLabels()
	for key, val := range v1.SyncerLabels() {
		if _, ok := ls[key]; !ok {
			continue
		}
		if ls[key] == val {
			return true
		}
	}

	return false
}
