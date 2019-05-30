package vet

import (
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// TODO(ekitson): Replace usage of this error with id.MultipleSingletonsError instead

// MultipleSingletonsErrorCode is the error code for MultipleSingletonsError
const MultipleSingletonsErrorCode = "1030"

func init() {
	rq1 := resourceQuota()
	rq1.Path = cmpath.FromSlash("namespaces/foo/rq1.yaml")
	rq1.MetaObject().SetName("quota-1")
	rq2 := resourceQuota()
	rq2.Path = cmpath.FromSlash("namespaces/foo/rq2.yaml")
	rq2.MetaObject().SetName("quota-2")
	status.AddExamples(MultipleSingletonsErrorCode, MultipleSingletonsError(
		rq1, rq2,
	))
}

var multipleSingletonsError = status.NewErrorBuilder(MultipleSingletonsErrorCode)

// MultipleSingletonsError reports that multiple singletons are defined in the same directory.
func MultipleSingletonsError(duplicates ...id.Resource) status.Error {
	var gvk schema.GroupVersionKind
	if len(duplicates) > 0 {
		gvk = duplicates[0].GroupVersionKind()
	}

	return multipleSingletonsError.WithResources(duplicates...).Errorf(
		"A directory may declare at most one %[1]q Resource:",
		kinds.ResourceString(gvk))
}
