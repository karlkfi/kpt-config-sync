package kptapplier

import (
	"github.com/google/nomos/pkg/status"
)

// ApplierErrorCode is the error code for apply failures.
const ApplierErrorCode = "2014"

var applierErrorBuilder = status.NewErrorBuilder(ApplierErrorCode)

// ApplierError indicates that the applier failed to apply some resources.
func ApplierError(err error) status.Error {
	return applierErrorBuilder.Wrap(err).Build()
}
