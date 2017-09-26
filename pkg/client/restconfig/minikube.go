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

const minikube = "minikube"

// NewMinikubeConfig creates a new configuration for connnecting to minikube from the kubectl
// config file on localhost.
func NewMinikubeConfig() (*rest.Config, error) {
	curentUser, err := user.Current()
	if err != nil {
		return nil, errors.Wrapf(err, "Faild to get current user")
	}

	configPath := filepath.Join(curentUser.HomeDir, kubectlConfigPath)
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: configPath}, &clientcmd.ConfigOverrides{})

	rawConfig, err := clientConfig.RawConfig()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get raw config for kubectl")
	}

	minikubeCluster, ok := rawConfig.Clusters[minikube]
	if !ok {
		return nil, errors.Wrapf(err, "Kubectl config did not have minikube Clusters entry")
	}
	minikubeAuthInfo, ok := rawConfig.AuthInfos[minikube]
	if !ok {
		return nil, errors.Wrapf(err, "Kubectl config did not have minikube AuthInfos entry")
	}

	return &rest.Config{
		Host: minikubeCluster.Server,
		TLSClientConfig: rest.TLSClientConfig{
			CertFile: minikubeAuthInfo.ClientCertificate,
			KeyFile:  minikubeAuthInfo.ClientKey,
			CAFile:   minikubeCluster.CertificateAuthority,
		},
	}, nil
}
