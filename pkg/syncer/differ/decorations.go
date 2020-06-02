package differ

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/hnc"
)

// enableManaged returns true if the resource explicitly has management enabled on a resource
// on the API server.
func managementEnabled(obj core.LabeledAndAnnotated) bool {
	return obj.GetAnnotations()[v1.ResourceManagementKey] == v1.ResourceManagementEnabled
}

// disableManaged returns true if the resource in the repo explicitly has management disabled.
func managementDisabled(obj core.LabeledAndAnnotated) bool {
	return obj.GetAnnotations()[v1.ResourceManagementKey] == v1.ResourceManagementDisabled
}

// managementUnset returns true if the resource has no Nomos ResourceManagementKey.
func managementUnset(obj core.LabeledAndAnnotated) bool {
	_, found := obj.GetAnnotations()[v1.ResourceManagementKey]
	return !found
}

// hasNomosMeta returns true if the given map has at least one Nomos annotation or label that syncer
// manages.
func hasNomosMeta(obj core.LabeledAndAnnotated) bool {
	as := obj.GetAnnotations()
	sas := append(v1.SyncerAnnotations(), hnc.AnnotationKey)
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
