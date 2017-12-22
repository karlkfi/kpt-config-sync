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

	"fmt"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"io/ioutil"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	certutil "k8s.io/client-go/util/cert"
	"os"
)

const kubectlConfigPath = ".kube/config"
const masterSecretsDir = "/etc/stolos/secrets/master/"

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

// NewLocalClusterConfig creates a config for connecting to the local cluster API server.
func NewLocalClusterConfig() (*rest.Config, error) {
	return rest.InClusterConfig()
}

// NewRemoteClusterConfig creates a config for connecting to a remote cluster API server.
func NewRemoteClusterConfig() (*rest.Config, error) {
	host := os.Getenv("REMOTE_KUBERNETES_API_SERVER_URL")
	if len(host) == 0 {
		return nil, fmt.Errorf("unable to load remote-cluster configuration, REMOTE_KUBERNETES_API_SERVER_URL must be defined")
	}

	token, err := ioutil.ReadFile(masterSecretsDir + v1.ServiceAccountTokenKey)
	if err != nil {
		return nil, err
	}
	tlsClientConfig := rest.TLSClientConfig{}
	rootCAFile := masterSecretsDir + v1.ServiceAccountRootCAKey
	if _, err := certutil.NewPool(rootCAFile); err != nil {
		glog.Errorf("Expected to load root CA config from %s, but got err: %v", rootCAFile, err)
	} else {
		tlsClientConfig.CAFile = rootCAFile
	}

	return &rest.Config{
		Host:            host,
		BearerToken:     string(token),
		TLSClientConfig: tlsClientConfig,
	}, nil
}
