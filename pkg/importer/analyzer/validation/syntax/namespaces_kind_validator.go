package syntax

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IllegalKindInNamespacesErrorCode is the error code for IllegalKindInNamespacesError
const IllegalKindInNamespacesErrorCode = "1038"

var illegalKindInNamespacesError = status.NewErrorBuilder(IllegalKindInNamespacesErrorCode)

// IllegalKindInNamespacesError reports that an object has been illegally defined in namespaces/
func IllegalKindInNamespacesError(resources ...client.Object) status.Error {
	return illegalKindInNamespacesError.
		Sprintf("Configs of the below Kind MUST not be declared in `%s`/:", repo.NamespacesDir).
		BuildWithResources(resources...)
}
