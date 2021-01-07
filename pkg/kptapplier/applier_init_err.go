package kptapplier

import (
	"github.com/google/nomos/pkg/status"
)

// ApplierInitErrorCode is the error code for initialization failure of the applier.
const ApplierInitErrorCode = "2009"

var applierInitErrorBuilder = status.NewErrorBuilder(ApplierInitErrorCode)

// ApplierInitError indicates that the applier is failed to be initialized
// due to the passed error.
func ApplierInitError(err error) status.Error {
	return applierInitErrorBuilder.Wrap(err).Build()
}
