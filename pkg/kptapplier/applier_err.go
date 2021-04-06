package kptapplier

import (
	"fmt"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/status"
)

// ApplierErrorCode is the error code for apply failures.
const ApplierErrorCode = "2009"

var applierErrorBuilder = status.NewErrorBuilder(ApplierErrorCode)

// ApplierError indicates that the applier failed to apply some resources.
func ApplierError(err error) status.Error {
	return applierErrorBuilder.Wrap(err).Build()
}

// ApplierErrorForResource indicates that the applier filed to apply
// the given resource.
func ApplierErrorForResource(err error, id core.ID) status.Error {
	return applierErrorBuilder.Wrap(fmt.Errorf("failed to apply %v: %w", id, err)).Build()
}
