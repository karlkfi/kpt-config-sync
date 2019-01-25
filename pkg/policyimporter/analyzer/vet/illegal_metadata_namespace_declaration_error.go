package vet

import "github.com/google/nomos/pkg/policyimporter/id"

// IllegalMetadataNamespaceDeclarationErrorCode is the error code for IllegalNamespaceDeclarationError
const IllegalMetadataNamespaceDeclarationErrorCode = "1009"

func init() {
	register(IllegalMetadataNamespaceDeclarationErrorCode, nil, "")
}

// IllegalMetadataNamespaceDeclarationError represents illegally declaring metadata.namespace
type IllegalMetadataNamespaceDeclarationError struct {
	id.Resource
}

// Error implements error.
func (e IllegalMetadataNamespaceDeclarationError) Error() string {
	// TODO(willbeason): Error unused until b/118715158
	return format(e,
		"Resources MUST NOT declare metadata.namespace:\n\n"+
			"%[1]s",
		id.PrintResource(e))
}

// Code implements Error
func (e IllegalMetadataNamespaceDeclarationError) Code() string {
	return IllegalMetadataNamespaceDeclarationErrorCode
}
