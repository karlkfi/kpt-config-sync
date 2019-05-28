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

var (
	// The function to use to get default current user.  Can be changed for tests
	// using SetCurrentUserForTest.
	userCurrentTestHook = defaultGetCurrentUser
	currentUser         = &user.User{}
)

func defaultGetCurrentUser() (*user.User, error) {
	return user.Current()
}

func customGetCurrentUser() (*user.User, error) {
	return currentUser, nil
}

// SetCurrentUserForTest sets the current user that will be returned, and/or
// the error to be reported.  This makes the tests independent of CGO for
// user.Current() that depend on CGO. Set the user to nil to revert to the
// default way of getting the current user.
func SetCurrentUserForTest(u *user.User) {
	if u == nil {
		userCurrentTestHook = defaultGetCurrentUser
		return
	}
	userCurrentTestHook = customGetCurrentUser
	currentUser = u
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
		restCfg, err := cfg.ClientConfig()
		if err != nil {
			badConfigs = append(badConfigs, fmt.Sprintf("%q: %v", ctxName, err))
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

// NewKubectlConfig creates a config for whichever context is active in kubectl.
func NewKubectlConfig() (*rest.Config, error) {
	if *flagKubectlContext != "" {
		return NewKubectlContextConfig(*flagKubectlContext)
	}

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

// NewKubectlContextConfig creates a new configuration for connnecting to kubernetes from the kubectl
// config file on localhost.
func NewKubectlContextConfig(contextName string) (*rest.Config, error) {
	clientConfig, err := NewClientConfigWithOverrides(
		&clientcmd.ConfigOverrides{
			CurrentContext: contextName,
		})
	if err != nil {
		return nil, errors.Wrapf(err, "NewKubectlContextConfig")
	}
	return clientConfig.ClientConfig()
}

// NewClientConfig returns the current (local) Kubernetes client configuration.
func NewClientConfig() (clientcmd.ClientConfig, error) {
	return NewClientConfigWithOverrides(&clientcmd.ConfigOverrides{})
}

// NewClientConfigWithOverrides returns a client configuration with supplied
// overrides.
func NewClientConfigWithOverrides(o *clientcmd.ConfigOverrides) (clientcmd.ClientConfig, error) {
	configPath, err := newConfigPath()
	if err != nil {
		return nil, errors.Wrapf(err, "while getting config path")
	}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: configPath}, o), nil
}

// NewLocalClusterConfig creates a config for connecting to the local cluster API server.
func NewLocalClusterConfig() (*rest.Config, error) {
	return rest.InClusterConfig()
}
