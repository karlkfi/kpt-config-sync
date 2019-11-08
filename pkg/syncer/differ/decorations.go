package differ

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
)

// enableManaged returns true if the resource explicitly has management enabled on a resource
// on the API server.
func managementEnabled(obj core.Annotated) bool {
	return obj.GetAnnotations()[v1.ResourceManagementKey] == v1.ResourceManagementEnabled
}

// disableManaged returns true if the resource in the repo explicitly has management disabled.
func managementDisabled(obj core.Annotated) bool {
	return obj.GetAnnotations()[v1.ResourceManagementKey] == v1.ResourceManagementDisabled
}

// managementUnset returns true if the resource has no Nomos ResourceManagementKey.
func managementUnset(obj core.Annotated) bool {
	_, found := obj.GetAnnotations()[v1.ResourceManagementKey]
	return !found
}

// hasNomosMeta returns true if the given map has at least one Nomos annotation or label that syncer
// manages.
func hasNomosMeta(obj core.LabeledAndAnnotated) bool {
	as := obj.GetAnnotations()
	for _, a := range v1.SyncerAnnotations() {
		if _, ok := as[a]; ok {
			return true
		}
	}
	ls := obj.GetLabels()
	for _, l := range v1.SyncerLabels() {
		if _, ok := ls[l]; ok {
			return true
		}
	}
	return false
}
