package veterrors

import (
	"sort"
	"strings"

	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
)

// MultipleNamespacesErrorCode is the error code for MultipleNamespacesError
const MultipleNamespacesErrorCode = "1030"

func init() {
	register(MultipleNamespacesErrorCode, nil, "")
}

// MultipleNamespacesError reports that multiple Namespaces are defined in the same directory.
type MultipleNamespacesError struct {
	Duplicates []ResourceID
}

// Error implements error
func (e MultipleNamespacesError) Error() string {
	var strs []string
	for _, duplicate := range e.Duplicates {
		strs = append(strs, printResourceID(duplicate))
	}
	sort.Strings(strs)

	return format(e,
		"A directory may declare at most one %[1]s Resource:\n\n"+
			"%[2]s",
		ast.Namespace, strings.Join(strs, "\n\n"))
}

// Code implements Error
func (e MultipleNamespacesError) Code() string { return MultipleNamespacesErrorCode }
