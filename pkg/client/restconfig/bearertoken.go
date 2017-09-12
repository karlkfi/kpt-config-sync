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
	"k8s.io/client-go/rest"
)

// NewBearerTokenConfig creates a rest.Config for connecting with a bearer token returned by running
// kubectl get secrets/[name] -oyaml.
// server - the address to the k8s API server
// caCertData - the base64 of the Certificate Authority Certificate (ca.crt in the yaml)
// bearerToken - The bearer token or the role account.  This is the Base64 decoded field 'token' from
// 	yaml file returned by kubectl when getting the secret.  Note that the JWT itself here is still in
// 	the "encoded" format (three dot-separated base64 encoded JSON objects).
func NewBearerTokenConfig(
	server string, caCertData []byte, bearerToken string) (*rest.Config, error) {
	return &rest.Config{
		Host:        server,
		BearerToken: bearerToken,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: caCertData,
		},
	}, nil
}
