package applier

import (
	"fmt"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/status"
)

// ApplierErrorCode is the error code for apply failures.
const ApplierErrorCode = "2009"

var applierErrorBuilder = status.NewErrorBuilder(ApplierErrorCode)

// Error indicates that the applier failed to apply some resources.
func Error(err error) status.Error {
	return applierErrorBuilder.Wrap(err).Build()
}

// ErrorForResource indicates that the applier filed to apply
// the given resource.
func ErrorForResource(err error, id core.ID) status.Error {
	return applierErrorBuilder.Wrap(fmt.Errorf("failed to apply %v: %w", id, err)).Build()
}
