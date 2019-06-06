package vet

import (
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// DeprecatedGroupKindErrorCode is the error code for DeprecatedGroupKindError.
const DeprecatedGroupKindErrorCode = "1050"

func init() {
	status.AddExamples(DeprecatedGroupKindErrorCode, DeprecatedGroupKindError(
		deprecatedDeployment(),
		kinds.Deployment()))
}

var deprecatedGroupKindError = status.NewErrorBuilder(DeprecatedGroupKindErrorCode)

// DeprecatedGroupKindError reports usage of a deprecated version of a specific Group/Kind.
func DeprecatedGroupKindError(resource id.Resource, expected schema.GroupVersionKind) status.Error {
	apiVersion, kind := expected.ToAPIVersionAndKind()
	return deprecatedGroupKindError.WithResources(resource).Errorf(
		"The config is using an unsupported Group and Kind. To fix, set the apiVersion to %q and kind to %q.",
		apiVersion, kind)
}
