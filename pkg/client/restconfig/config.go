/*
Copyright 2017 The Nomos Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package restconfig

import (
	"os"
	"os/user"
	"path/filepath"

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
