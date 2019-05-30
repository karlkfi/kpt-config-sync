package vet

import (
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// IllegalMetadataNamespaceDeclarationErrorCode is the error code for IllegalNamespaceDeclarationError
const IllegalMetadataNamespaceDeclarationErrorCode = "1009"

func init() {
	status.AddExamples(IllegalMetadataNamespaceDeclarationErrorCode, IllegalMetadataNamespaceDeclarationError(
		role(),
		"foo",
	))
}

var illegalMetadataNamespaceDeclarationError = status.NewErrorBuilder(IllegalMetadataNamespaceDeclarationErrorCode)

// IllegalMetadataNamespaceDeclarationError represents illegally declaring metadata.namespace
func IllegalMetadataNamespaceDeclarationError(resource id.Resource, expectedNamespace string) status.Error {
	return illegalMetadataNamespaceDeclarationError.WithResources(resource).Errorf(
		"A config MUST either declare a `metadata.namespace` field exactly matching the directory "+
			"containing the config, %q, or leave the field blank:",
		expectedNamespace)
}
