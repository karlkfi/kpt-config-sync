package veterrors

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
)

// IllegalTopLevelNamespaceError reports that there may not be a Namespace declared directly in namespaces/
type IllegalTopLevelNamespaceError struct {
	ResourceID
}

// Error implements error
func (e IllegalTopLevelNamespaceError) Error() string {
	return format(e,
		"%[2]ss MUST be declared in subdirectories of %[1]s/. Create a subdirectory for %[2]ss declared in:\n\n"+
			"%[3]s",
		repo.NamespacesDir, ast.Namespace, printResourceID(e))
}

// Code implements Error
func (e IllegalTopLevelNamespaceError) Code() string { return IllegalTopLevelNamespaceErrorCode }
