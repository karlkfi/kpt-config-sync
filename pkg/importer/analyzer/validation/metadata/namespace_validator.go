package metadata

import (
	"github.com/google/nomos/pkg/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IllegalMetadataNamespaceDeclarationErrorCode is the error code for IllegalNamespaceDeclarationError
const IllegalMetadataNamespaceDeclarationErrorCode = "1009"

var illegalMetadataNamespaceDeclarationError = status.NewErrorBuilder(IllegalMetadataNamespaceDeclarationErrorCode)

// IllegalMetadataNamespaceDeclarationError represents illegally declaring metadata.namespace
func IllegalMetadataNamespaceDeclarationError(resource client.Object, expectedNamespace string) status.Error {
	return illegalMetadataNamespaceDeclarationError.
		Sprintf("A config MUST either declare a `namespace` field exactly matching the directory "+
			"containing the config, %q, or leave the field blank:", expectedNamespace).
		BuildWithResources(resource)
}
