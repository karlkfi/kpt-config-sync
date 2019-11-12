package filesystem

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/status"
)

// ConfigManagementNotInstalledErrorCode is the error code for ConfigManagementNotInstalledError
const ConfigManagementNotInstalledErrorCode = "1016"

var configManagementNotInstalledError = status.NewErrorBuilder(ConfigManagementNotInstalledErrorCode)

// ConfigManagementNotInstalledError reports that Nomos has not been installed properly.
func ConfigManagementNotInstalledError(err error) status.Error {
	return configManagementNotInstalledError.Wrapf(err, "%s is not properly installed. Apply a %s config to enable config management.",
		configmanagement.ProductName, configmanagement.OperatorKind)
}
