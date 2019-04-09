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
	status.Register(MultipleSingletonsErrorCode, MultipleSingletonsError{
		Duplicates: []id.Resource{
			rq1, rq2,
		},
	})
}

// MultipleSingletonsError reports that multiple singletons are defined in the same directory.
type MultipleSingletonsError struct {
	Duplicates []id.Resource
}

var _ status.ResourceError = &MultipleSingletonsError{}

// Error implements error
func (e MultipleSingletonsError) Error() string {
	var gvk schema.GroupVersionKind
	if len(e.Duplicates) > 0 {
		gvk = e.Duplicates[0].GroupVersionKind()
	}

	return status.Format(e,
		"A directory may declare at most one %[1]q Resource:",
		kinds.ResourceString(gvk))
}

// Code implements Error
func (e MultipleSingletonsError) Code() string { return MultipleSingletonsErrorCode }

// Resources implements ResourceError
func (e MultipleSingletonsError) Resources() []id.Resource {
	return e.Duplicates
}
