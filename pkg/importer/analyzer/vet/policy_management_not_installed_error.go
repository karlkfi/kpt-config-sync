package vet

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
)

// PolicyManagementNotInstalledErrorCode is the error code for PolicyManagementNotInstalledError
const PolicyManagementNotInstalledErrorCode = "1016"

func init() {
	status.Register(PolicyManagementNotInstalledErrorCode, PolicyManagementNotInstalledError{
		Err: errors.New("cluster doesn't have required CRD"),
	})
}

var _ status.Error = PolicyManagementNotInstalledError{}

// PolicyManagementNotInstalledError reports that Nomos has not been installed properly.
type PolicyManagementNotInstalledError struct {
	Err error
}

// Error implements error.
func (e PolicyManagementNotInstalledError) Error() string {
	return status.Format(e, errors.Wrapf(e.Err, "%s is not properly installed. Apply a %s config to enable config management.",
		configmanagement.ProductName, configmanagement.OperatorKind).Error())
}

// Code implements Error.
func (e PolicyManagementNotInstalledError) Code() string {
	return PolicyManagementNotInstalledErrorCode
}

// ToCME implements ToCMEr.
func (e PolicyManagementNotInstalledError) ToCME() v1.ConfigManagementError {
	return status.FromError(e)
}
