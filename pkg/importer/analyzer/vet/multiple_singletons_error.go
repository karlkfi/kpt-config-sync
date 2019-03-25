package vet

import (
	"sort"
	"strings"

	"github.com/google/nomos/pkg/kinds"

	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/id"
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

var _ id.ResourceError = &MultipleSingletonsError{}

// Error implements error
func (e MultipleSingletonsError) Error() string {
	var strs []string
	var gvk schema.GroupVersionKind
	for _, duplicate := range e.Duplicates {
		strs = append(strs, id.PrintResource(duplicate))
		gvk = duplicate.GroupVersionKind()
	}
	sort.Strings(strs)

	return status.Format(e,
		"A directory may declare at most one %[1]q Resource:\n\n"+
			"%[2]s",
		kinds.ResourceString(gvk), strings.Join(strs, "\n\n"))
}

// Code implements Error
func (e MultipleSingletonsError) Code() string { return MultipleSingletonsErrorCode }

// Resources implements ResourceError
func (e MultipleSingletonsError) Resources() []id.Resource {
	return e.Duplicates
}
