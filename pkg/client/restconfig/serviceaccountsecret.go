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
	"encoding/base64"
	"io/ioutil"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/rest"
)

// Secret redefined to work around dependency issue on on yaml.v2 since k8s has it
// vendored at the moment.  This should eventually use k8s.io/api/core/v1 for the
// definition of secret.
type Secret struct {
	Data struct {
		CaCrtB64 string `yaml:"ca.crt"`
		TokenB64 string `yaml:"token"`
	}
}

// NewConfigFromSecret creates a new rest client config from a secret produced by kubectl get secrets/[name]
// and a server address to connnect to.
func NewConfigFromSecret(serverAddress, secretPath string) (*rest.Config, error) {
	fileBytes, err := ioutil.ReadFile(secretPath)
	var secret Secret
	err = yaml.Unmarshal(fileBytes, &secret)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to unmarshal secret")
	}

	caCertData, err := base64.StdEncoding.DecodeString(secret.Data.CaCrtB64)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to decode ca crt base64")
	}

	jsonWebTokenBytes, err := base64.StdEncoding.DecodeString(secret.Data.TokenB64)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to decode token base64")
	}

	return NewBearerTokenConfig(serverAddress, caCertData, string(jsonWebTokenBytes))
}
