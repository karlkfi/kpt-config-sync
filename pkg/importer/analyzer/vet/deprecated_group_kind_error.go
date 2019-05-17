package vet

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// DeprecatedGroupKindErrorCode is the error code for DeprecatedGroupKindError.
const DeprecatedGroupKindErrorCode = "1050"

func init() {
	status.Register(DeprecatedGroupKindErrorCode, DeprecatedGroupKindError{
		Resource: deprecatedDeployment(),
		Expected: kinds.Deployment().GroupKind(),
	})
}

// DeprecatedGroupKindError reports a config with a deprecated GroupKind in the repo.
type DeprecatedGroupKindError struct {
	id.Resource
	Expected schema.GroupKind
}

var _ status.ResourceError = DeprecatedGroupKindError{}

// Error implements error.
func (e DeprecatedGroupKindError) Error() string {
	return status.Format(e,
		"The config is using an unsupported Group and Kind. To fix, change the config to have a Group and Kind of: %q",
		e.Expected)
}

// Code implements Error
func (e DeprecatedGroupKindError) Code() string { return DeprecatedGroupKindErrorCode }

// Resources implements ResourceError
func (e DeprecatedGroupKindError) Resources() []id.Resource {
	return []id.Resource{e.Resource}
}

// ToCME implements ToCMEr.
func (e DeprecatedGroupKindError) ToCME() v1.ConfigManagementError {
	return status.FromResourceError(e)
}
