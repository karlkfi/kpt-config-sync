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

// Package client provides a way to set up a client to a kubernetes cluster.  At the moment, it serves as an example
// for reading the kubectl config and then connecting to minikube.
package client

import (
	"io/ioutil"
	"os/user"
	"path/filepath"

	"github.com/mdruskin/kubernetes-enterprise-control/pkg/client/restconfig"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/clientcmd/api/v1"
)

const (
	kubectlConfigPath = ".kube/config"
	minikube          = "minikube"
)

// NewClient creates a new client of a given type.
func NewClient(clientType string) (*Client, error) {
	switch clientType {
	case minikube:
		return NewMiniKubeClient()
	}
	return nil, errors.Errorf("Invalid client type %s", clientType)
}

func NewServiceAccountClient(secretPath string) (*Client, error) {
	kubectlConfig, err := loadKubectlConfig()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to load kubectl config")
	}

	minikubeCluster, ok := kubectlConfig.Clusters[minikube]
	if !ok {
		return nil, errors.Wrapf(err, "Kubectl config did not have minikube Clusters entry")
	}

	cfg, err := restconfig.NewConfigFromSecret(minikubeCluster.Server, secretPath)
	if err != nil {
		return nil, err
	}

	clientSet, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create kubernetes client")
	}
	if clientSet == nil {
		return nil, errors.Errorf("Got nil client with nil error")
	}

	return &Client{clientSet: clientSet}, nil
}

func loadKubectlConfig() (*api.Config, error) {
	curentUser, err := user.Current()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get current user")
	}

	fileBytes, err := ioutil.ReadFile(filepath.Join(curentUser.HomeDir, kubectlConfigPath))
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to read kubectl config")
	}

	scheme := runtime.NewScheme()
	v1.AddToScheme(scheme)
	api.AddToScheme(scheme)
	codecFactory := serializer.NewCodecFactory(scheme)

	config := &api.Config{}
	loadedConfig, _, err := codecFactory.UniversalDecoder(api.SchemeGroupVersion).Decode(fileBytes, nil, config)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to deserialize using codec factory")
	}

	return loadedConfig.(*api.Config), nil
}

// NewMiniKubeClient creates a client that will talk to minikube
func NewMiniKubeClient() (*Client, error) {
	kubectlConfig, err := loadKubectlConfig()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to load kubectl config")
	}

	minikubeCluster, ok := kubectlConfig.Clusters[minikube]
	if !ok {
		return nil, errors.Wrapf(err, "Kubectl config did not have minikube Clusters entry")
	}
	minikubeAuthInfo, ok := kubectlConfig.AuthInfos[minikube]
	if !ok {
		return nil, errors.Wrapf(err, "Kubectl config did not have minikube AuthInfos entry")
	}

	cfg := &rest.Config{
		Host: minikubeCluster.Server,
		TLSClientConfig: rest.TLSClientConfig{
			CertFile: minikubeAuthInfo.ClientCertificate,
			KeyFile:  minikubeAuthInfo.ClientKey,
			CAFile:   minikubeCluster.CertificateAuthority,
		},
	}

	clientSet, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create kubernetes client")
	}
	if clientSet == nil {
		return nil, errors.Errorf("Got nil client with nil error")
	}

	return &Client{clientSet: clientSet}, nil
}
