package veterrors

import "github.com/google/nomos/pkg/policyimporter/analyzer/ast"

// IllegalAbstractNamespaceObjectKindErrorCode is the error code for IllegalAbstractNamespaceObjectKindError
const IllegalAbstractNamespaceObjectKindErrorCode = "1007"

func init() {
	register(IllegalAbstractNamespaceObjectKindErrorCode, nil, "")
}

// IllegalAbstractNamespaceObjectKindError represents an illegal usage of a kind not allowed in abstract namespaces.
type IllegalAbstractNamespaceObjectKindError struct {
	ResourceID
}

// Error implements error.
func (e IllegalAbstractNamespaceObjectKindError) Error() string {
	return format(e,
		"Resource %[4]q illegally declared in an %[1]s directory. "+
			"Move this Resource to a %[2]s directory:\n\n"+
			"%[3]s",
		ast.AbstractNamespace, ast.Namespace, printResourceID(e), e.Name())
}

// Code implements Error
func (e IllegalAbstractNamespaceObjectKindError) Code() string {
	return IllegalAbstractNamespaceObjectKindErrorCode
}
