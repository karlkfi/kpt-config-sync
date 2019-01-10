package veterrors

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/id"
)

// IllegalNamespaceSelectorAnnotationErrorCode is the error code for IllegalNamespaceSelectorAnnotationError
const IllegalNamespaceSelectorAnnotationErrorCode = "1004"

func init() {
	register(IllegalNamespaceSelectorAnnotationErrorCode, nil, "")
}

// IllegalNamespaceSelectorAnnotationError represents an illegal usage of the namespace selector annotation.
type IllegalNamespaceSelectorAnnotationError struct {
	id.TreeNode
}

// Error implements error.
func (e IllegalNamespaceSelectorAnnotationError) Error() string {
	return format(e,
		"A %[3]s MUST NOT use the annotation %[2]s. "+
			"Remove metadata.annotations.%[2]s from:\n\n"+
			"%[1]s",
		id.PrintTreeNode(e.TreeNode), v1alpha1.NamespaceSelectorAnnotationKey, ast.Namespace)
}

// Code implements Error
func (e IllegalNamespaceSelectorAnnotationError) Code() string {
	return IllegalNamespaceSelectorAnnotationErrorCode
}
