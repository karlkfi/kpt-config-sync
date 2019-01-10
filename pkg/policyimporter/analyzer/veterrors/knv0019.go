package veterrors

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/id"
)

// IllegalTopLevelNamespaceErrorCode is the error code for IllegalTopLevelNamespaceError
const IllegalTopLevelNamespaceErrorCode = "1019"

func init() {
	register(IllegalTopLevelNamespaceErrorCode, nil, "")
}

// IllegalTopLevelNamespaceError reports that there may not be a Namespace declared directly in namespaces/
type IllegalTopLevelNamespaceError struct {
	id.Resource
}

// Error implements error
func (e IllegalTopLevelNamespaceError) Error() string {
	return format(e,
		"%[2]ss MUST be declared in subdirectories of %[1]s/. Create a subdirectory for %[2]ss declared in:\n\n"+
			"%[3]s",
		repo.NamespacesDir, ast.Namespace, id.PrintResource(e))
}

// Code implements Error
func (e IllegalTopLevelNamespaceError) Code() string { return IllegalTopLevelNamespaceErrorCode }
