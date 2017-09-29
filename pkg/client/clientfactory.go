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
	"github.com/google/stolos/pkg/client/meta"
	"github.com/google/stolos/pkg/client/restconfig"
	"github.com/pkg/errors"
)

const minikube = "minikube"

// NewClient creates a new client of a given type.
func NewClient(clientType string) (*meta.Client, error) {
	switch clientType {
	case minikube:
		return NewMiniKubeClient()
	}
	return nil, errors.Errorf("Invalid client type %s", clientType)
}

// NewMinikubeServiceAccountClient returns a new client for a minikube service account.
func NewMinikubeServiceAccountClient(secretPath string) (*meta.Client, error) {
	minikubeConfig, err := restconfig.NewMinikubeConfig()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to load kubectl config")
	}

	cfg, err := restconfig.NewConfigFromSecret(minikubeConfig.Host, secretPath)
	if err != nil {
		return nil, err
	}

	return meta.NewForConfig(cfg)
}

// NewMiniKubeClient creates a client that will talk to minikube
func NewMiniKubeClient() (*meta.Client, error) {
	cfg, err := restconfig.NewKubectlConfig()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to load kubectl config")
	}

	return meta.NewForConfig(cfg)
}
