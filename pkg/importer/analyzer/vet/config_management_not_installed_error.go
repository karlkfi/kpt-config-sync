package vet

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
)

// ConfigManagementNotInstalledErrorCode is the error code for ConfigManagementNotInstalledError
const ConfigManagementNotInstalledErrorCode = "1016"

func init() {
	status.AddExamples(ConfigManagementNotInstalledErrorCode, ConfigManagementNotInstalledError(errors.New("cluster doesn't have required CRD")))
}

var configManagementNotInstalledError = status.NewErrorBuilder(ConfigManagementNotInstalledErrorCode)

// ConfigManagementNotInstalledError reports that Nomos has not been installed properly.
func ConfigManagementNotInstalledError(err error) status.Error {
	return configManagementNotInstalledError.Wrapf(err, "%s is not properly installed. Apply a %s config to enable config management.",
		configmanagement.ProductName, configmanagement.OperatorKind)
}
