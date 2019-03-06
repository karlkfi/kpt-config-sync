package vet

import (
	"github.com/google/nomos/pkg/policyimporter/id"
	"github.com/google/nomos/pkg/status"
)

// IllegalMetadataNamespaceDeclarationErrorCode is the error code for IllegalNamespaceDeclarationError
const IllegalMetadataNamespaceDeclarationErrorCode = "1009"

func init() {
	register(IllegalMetadataNamespaceDeclarationErrorCode)
}

// IllegalMetadataNamespaceDeclarationError represents illegally declaring metadata.namespace
type IllegalMetadataNamespaceDeclarationError struct {
	id.Resource
}

// Error implements error.
func (e IllegalMetadataNamespaceDeclarationError) Error() string {
	return status.Format(e,
		"A config MUST either declare a metadata.namespace field exactly matching the directory "+
			"containing the config, %[1]q, or leave the field blank:\n\n"+
			"%[2]s",
		e.Resource.Dir().Base(), id.PrintResource(e))
}

// Code implements Error
func (e IllegalMetadataNamespaceDeclarationError) Code() string {
	return IllegalMetadataNamespaceDeclarationErrorCode
}
