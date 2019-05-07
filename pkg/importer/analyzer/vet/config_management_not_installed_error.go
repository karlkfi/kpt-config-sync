package vet

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
)

// ConfigManagementNotInstalledErrorCode is the error code for ConfigManagementNotInstalledError
const ConfigManagementNotInstalledErrorCode = "1016"

func init() {
	status.Register(ConfigManagementNotInstalledErrorCode, ConfigManagementNotInstalledError{
		Err: errors.New("cluster doesn't have required CRD"),
	})
}

var _ status.Error = ConfigManagementNotInstalledError{}

// ConfigManagementNotInstalledError reports that Nomos has not been installed properly.
type ConfigManagementNotInstalledError struct {
	Err error
}

// Error implements error.
func (e ConfigManagementNotInstalledError) Error() string {
	return status.Format(e, errors.Wrapf(e.Err, "%s is not properly installed. Apply a %s config to enable config management.",
		configmanagement.ProductName, configmanagement.OperatorKind).Error())
}

// Code implements Error.
func (e ConfigManagementNotInstalledError) Code() string {
	return ConfigManagementNotInstalledErrorCode
}

// ToCME implements ToCMEr.
func (e ConfigManagementNotInstalledError) ToCME() v1.ConfigManagementError {
	return status.FromError(e)
}
