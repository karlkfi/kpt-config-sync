package vet

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// IllegalMetadataNamespaceDeclarationErrorCode is the error code for IllegalNamespaceDeclarationError
const IllegalMetadataNamespaceDeclarationErrorCode = "1009"

func init() {
	status.Register(IllegalMetadataNamespaceDeclarationErrorCode, IllegalMetadataNamespaceDeclarationError{
		Resource:          role(),
		ExpectedNamespace: "foo",
	})
}

// IllegalMetadataNamespaceDeclarationError represents illegally declaring metadata.namespace
type IllegalMetadataNamespaceDeclarationError struct {
	id.Resource
	ExpectedNamespace string
}

var _ status.ResourceError = IllegalMetadataNamespaceDeclarationError{}

// Error implements error.
func (e IllegalMetadataNamespaceDeclarationError) Error() string {
	return status.Format(e,
		"A config MUST either declare a `metadata.namespace` field exactly matching the directory "+
			"containing the config, %q, or leave the field blank:",
		e.ExpectedNamespace)
}

// Code implements Error
func (e IllegalMetadataNamespaceDeclarationError) Code() string {
	return IllegalMetadataNamespaceDeclarationErrorCode
}

// Resources implements ResourceError
func (e IllegalMetadataNamespaceDeclarationError) Resources() []id.Resource {
	return []id.Resource{e.Resource}
}

// ToCME implements ToCMEr.
func (e IllegalMetadataNamespaceDeclarationError) ToCME() v1.ConfigManagementError {
	return status.FromResourceError(e)
}
