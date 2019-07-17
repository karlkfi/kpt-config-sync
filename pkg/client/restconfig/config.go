package restconfig

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const kubectlConfigPath = ".kube/config"

// The function to use to get default current user.  Can be changed for tests
// using SetCurrentUserForTest.
var userCurrentTestHook = defaultGetCurrentUser

func defaultGetCurrentUser() (*user.User, error) {
	return user.Current()
}

// newConfigPath returns the correct kubeconfig file path to use, depending on
// the current user settings and the runtime environment.
func newConfigPath() (string, error) {
	// First try the KUBECONFIG variable.
	envPath := os.Getenv("KUBECONFIG")
	if envPath != "" {
		return envPath, nil
	}
	// Try the current user.
	curentUser, err := userCurrentTestHook()
	if err != nil {
		return "", errors.Wrapf(err, "failed to get current user")
	}
	path := filepath.Join(curentUser.HomeDir, kubectlConfigPath)
	return path, nil
}

// newConfigFromPath creates a rest.Config from a configuration file at the
// supplied path.
func newConfigFromPath(path string) (*rest.Config, error) {
	config, err := clientcmd.BuildConfigFromFlags("", path)
	if err != nil {
		return nil, err
	}
	return config, nil
}

// AllKubectlConfigs creates a config for every context available in the kubeconfig. The configs are
// mapped by context name. There is no way to detect unhealthy clusters specified by a context, so
// timeout can be used to prevent calls to those clusters from hanging for long periods of time.
func AllKubectlConfigs(timeout time.Duration) (map[string]*rest.Config, error) {
	configPath, err := newConfigPath()
	if err != nil {
		return nil, errors.Wrap(err, "while getting config path")
	}

	rules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: configPath}
	clientCfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, &clientcmd.ConfigOverrides{})
	apiCfg, err := clientCfg.RawConfig()
	if err != nil {
		return nil, errors.Wrap(err, "while building client config")
	}

	var badConfigs []string
	configs := map[string]*rest.Config{}
	for ctxName := range apiCfg.Contexts {
		cfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			rules, &clientcmd.ConfigOverrides{CurrentContext: ctxName})
		restCfg, err2 := cfg.ClientConfig()
		if err2 != nil {
			badConfigs = append(badConfigs, fmt.Sprintf("%q: %v", ctxName, err2))
			continue
		}

		if timeout > 0 {
			restCfg.Timeout = timeout
		}
		configs[ctxName] = restCfg
	}

	var cfgErrs error
	if len(badConfigs) > 0 {
		cfgErrs = fmt.Errorf("failed to build configs:\n%s", strings.Join(badConfigs, "\n"))
	}
	return configs, cfgErrs
}

// newKubectlConfig creates a config for whichever context is active in kubectl.
func newKubectlConfig() (*rest.Config, error) {
	path, err := newConfigPath()
	if err != nil {
		return nil, errors.Wrapf(err, "while getting config path")
	}
	config, err := newConfigFromPath(path)
	if err != nil {
		return nil, errors.Wrapf(err, "while loading from %v", path)
	}
	return config, nil
}

// NewClientConfig returns the current (local) Kubernetes client configuration.
func NewClientConfig() (clientcmd.ClientConfig, error) {
	return newClientConfigWithOverrides(&clientcmd.ConfigOverrides{})
}

// newClientConfigWithOverrides returns a client configuration with supplied
// overrides.
func newClientConfigWithOverrides(o *clientcmd.ConfigOverrides) (clientcmd.ClientConfig, error) {
	configPath, err := newConfigPath()
	if err != nil {
		return nil, errors.Wrapf(err, "while getting config path")
	}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: configPath}, o), nil
}

// newLocalClusterConfig creates a config for connecting to the local cluster API server.
func newLocalClusterConfig() (*rest.Config, error) {
	return rest.InClusterConfig()
}
