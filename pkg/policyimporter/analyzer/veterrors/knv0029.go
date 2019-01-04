package veterrors

import (
	"sort"
	"strings"

	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
)

// MetadataNameCollisionError reports that multiple objects in the same namespace of the same Kind share a name.
type MetadataNameCollisionError struct {
	Name       string
	Duplicates []ResourceID
}

// Error implements error
func (e MetadataNameCollisionError) Error() string {
	var strs []string
	for _, duplicate := range e.Duplicates {
		strs = append(strs, printResourceID(duplicate))
	}
	sort.Strings(strs)

	return format(e,
		"Resources of the same Kind MUST have unique names in the same %[1]s and their parent %[3]ss:\n\n"+
			"%[2]s",
		ast.Namespace, strings.Join(strs, "\n\n"), ast.AbstractNamespace)
}

// Code implements Error
func (e MetadataNameCollisionError) Code() string { return MetadataNameCollisionErrorCode }
