package applier

import (
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ManagementConflictErrorCode is the error code for management conflict errors.
const ManagementConflictErrorCode = "1060"

var managementConflictErrorBuilder = status.NewErrorBuilder(ManagementConflictErrorCode)

// ManagementConflictError indicates that the passed resource is illegally
// declared in both the Root repository and a Namespace repository.
func ManagementConflictError(resource client.Object) status.Error {
	return managementConflictErrorBuilder.
		Sprintf("The %q reconciler cannot manage resources declared in the Root repository. "+
			"Remove the declaration for this resource from either the Namespace repository, or the Root repository.",
			resource.GetAnnotations()[v1alpha1.ResourceManagerKey]).
		BuildWithResources(resource)
}
