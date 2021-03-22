package nonhierarchical

import (
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DeprecatedGroupKindErrorCode is the error code for DeprecatedGroupKindError.
const DeprecatedGroupKindErrorCode = "1050"

var deprecatedGroupKindError = status.NewErrorBuilder(DeprecatedGroupKindErrorCode)

// DeprecatedGroupKindError reports usage of a deprecated version of a specific Group/Kind.
func DeprecatedGroupKindError(resource client.Object, expected schema.GroupVersionKind) status.Error {
	return deprecatedGroupKindError.
		Sprintf("The config is using a deprecated Group and Kind. To fix, set the Group and Kind to %q",
			expected.GroupKind().String()).
		BuildWithResources(resource)
}
