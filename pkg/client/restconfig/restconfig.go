package restconfig

import (
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // kubectl auth provider plugins
	"k8s.io/client-go/rest"
)

// A source for creating a rest config
type configSource struct {
	name   string                       // The name for the config
	create func() (*rest.Config, error) // The function for creating the config
}

// List of config sources that will be tried in order for creating a rest.Config
var configSources = []configSource{
	{
		name:   "podServiceAccount",
		create: newLocalClusterConfig,
	},
	{
		name:   "kubectl",
		create: newKubectlConfig,
	},
}

// NewRestConfig will attempt to create a new rest config from all configured options and return
// the first successfully created configuration.  The flag restConfigSource, if specified, will
// change the behvior to attempt to create from only the configured source.
func NewRestConfig() (*rest.Config, error) {
	var errorStrs []string
	for _, source := range configSources {
		config, err := source.create()
		if err == nil {
			glog.V(1).Infof("Created rest config from source %s", source.name)
			glog.V(7).Infof("Config: %#v", *config)
			return config, nil
		}
		glog.V(5).Infof("Failed to create from %s: %s", source.name, err)
		errorStrs = append(errorStrs, fmt.Sprintf("%s: %s", source.name, err))
	}

	return nil, errors.Errorf("Unable to create rest config:\n%s", strings.Join(errorStrs, "\n"))
}
