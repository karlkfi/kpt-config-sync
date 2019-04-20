package vet

import (
	"strings"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// IllegalAbstractNamespaceObjectKindErrorCode is the error code for IllegalAbstractNamespaceObjectKindError
const IllegalAbstractNamespaceObjectKindErrorCode = "1007"

func init() {
	status.Register(IllegalAbstractNamespaceObjectKindErrorCode, IllegalAbstractNamespaceObjectKindError{
		Resource: role(),
	})
}

// IllegalAbstractNamespaceObjectKindError represents an illegal usage of a kind not allowed in abstract namespaces.
// TODO(willbeason): Consolidate Illegal{X}ObjectKindErrors
type IllegalAbstractNamespaceObjectKindError struct {
	id.Resource
}

var _ status.ResourceError = IllegalAbstractNamespaceObjectKindError{}

// Error implements error.
func (e IllegalAbstractNamespaceObjectKindError) Error() string {
	return status.Format(e,
		"Config `%[3]s` illegally declared in an %[1]s directory. "+
			"Move this config to a %[2]s directory:",
		strings.ToLower(string(node.AbstractNamespace)), node.Namespace, e.Name())
}

// Code implements Error
func (e IllegalAbstractNamespaceObjectKindError) Code() string {
	return IllegalAbstractNamespaceObjectKindErrorCode
}

// Resources implements ResourceError
func (e IllegalAbstractNamespaceObjectKindError) Resources() []id.Resource {
	return []id.Resource{e.Resource}
}

// ToCME implements ToCMEr.
func (e IllegalAbstractNamespaceObjectKindError) ToCME() v1.ConfigManagementError {
	return status.FromResourceError(e)
}
