/*
Copyright 2018 The Nomos Authors.

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

package v1alpha1

import (
	"github.com/google/nomos/clientgen/apis/scheme"
	v1alpha1 "github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer"
	rest "k8s.io/client-go/rest"
)

type NomosV1alpha1Interface interface {
	RESTClient() rest.Interface
	ClusterSelectorsGetter
	NamespaceSelectorsGetter
	ReposGetter
	SyncsGetter
}

// NomosV1alpha1Client is used to interact with features provided by the nomos.dev group.
type NomosV1alpha1Client struct {
	restClient rest.Interface
}

func (c *NomosV1alpha1Client) ClusterSelectors() ClusterSelectorInterface {
	return newClusterSelectors(c)
}

func (c *NomosV1alpha1Client) NamespaceSelectors() NamespaceSelectorInterface {
	return newNamespaceSelectors(c)
}

func (c *NomosV1alpha1Client) Repos() RepoInterface {
	return newRepos(c)
}

func (c *NomosV1alpha1Client) Syncs() SyncInterface {
	return newSyncs(c)
}

// NewForConfig creates a new NomosV1alpha1Client for the given config.
func NewForConfig(c *rest.Config) (*NomosV1alpha1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &NomosV1alpha1Client{client}, nil
}

// NewForConfigOrDie creates a new NomosV1alpha1Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *NomosV1alpha1Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new NomosV1alpha1Client for the given RESTClient.
func New(c rest.Interface) *NomosV1alpha1Client {
	return &NomosV1alpha1Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv := v1alpha1.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *NomosV1alpha1Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
