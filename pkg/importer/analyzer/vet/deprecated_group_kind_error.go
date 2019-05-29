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
		kinds.Deployment().GroupKind()))
}

var deprecatedGroupKindError = status.NewErrorBuilder(DeprecatedGroupKindErrorCode)

// DeprecatedGroupKindError reports usage of a deprecated version of a specific Group/Kind.
func DeprecatedGroupKindError(resource id.Resource, expected schema.GroupKind) status.Error {
	return deprecatedGroupKindError.WithResources(resource).Errorf(
		"The config is using an unsupported Group and Kind. To fix, change the config to have a Group and Kind of: %q",
		expected)
}
