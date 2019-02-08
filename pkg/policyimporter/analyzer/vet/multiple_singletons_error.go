package vet

import (
	"sort"
	"strings"

	"github.com/google/nomos/pkg/policyimporter/id"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// MultipleSingletonsErrorCode is the error code for MultipleSingletonsError
const MultipleSingletonsErrorCode = "1030"

func init() {
	register(MultipleSingletonsErrorCode, nil, "")
}

// MultipleSingletonsError reports that multiple singletons are defined in the same directory.
type MultipleSingletonsError struct {
	Duplicates []id.Resource
}

// Error implements error
func (e MultipleSingletonsError) Error() string {
	var strs []string
	var gvk schema.GroupVersionKind
	for _, duplicate := range e.Duplicates {
		strs = append(strs, id.PrintResource(duplicate))
		gvk = duplicate.GroupVersionKind()
	}
	sort.Strings(strs)

	return format(e,
		"A directory may declare at most one %[1]q Resource:\n\n"+
			"%[2]s",
		gvk.String(), strings.Join(strs, "\n\n"))
}

// Code implements Error
func (e MultipleSingletonsError) Code() string { return MultipleSingletonsErrorCode }
