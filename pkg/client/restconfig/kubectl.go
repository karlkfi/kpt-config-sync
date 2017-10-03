/*
Copyright 2017 The Kubernetes Authors.
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
	"os/user"
	"path/filepath"

	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const kubectlConfigPath = ".kube/config"

// NewKubectlConfig creates a config for whichever context is active in kubectl.
func NewKubectlConfig() (*rest.Config, error) {
	if *flagKubectlContext != "" {
		return NewKubectlContextConfig(*flagKubectlContext)
	}

	curentUser, err := user.Current()
	if err != nil {
		return nil, errors.Wrapf(err, "Faild to get current user")
	}

	config, err := clientcmd.BuildConfigFromFlags("", filepath.Join(curentUser.HomeDir, kubectlConfigPath))
	if err != nil {
		return nil, err
	}
	return config, nil
}

// NewKubectlContextConfig creates a new configuration for connnecting to kubernetes from the kubectl
// config file on localhost.
func NewKubectlContextConfig(contextName string) (*rest.Config, error) {
	curentUser, err := user.Current()
	if err != nil {
		return nil, errors.Wrapf(err, "Faild to get current user")
	}

	configPath := filepath.Join(curentUser.HomeDir, kubectlConfigPath)
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: configPath},
		&clientcmd.ConfigOverrides{
			CurrentContext: contextName,
		})
	return clientConfig.ClientConfig()
}
