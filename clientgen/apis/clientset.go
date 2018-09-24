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

package apis

import (
	glog "github.com/golang/glog"
	bespinv1 "github.com/google/nomos/clientgen/apis/typed/policyascode/v1"
	nomosv1 "github.com/google/nomos/clientgen/apis/typed/policyhierarchy/v1"
	nomosv1alpha1 "github.com/google/nomos/clientgen/apis/typed/policyhierarchy/v1alpha1"
	discovery "k8s.io/client-go/discovery"
	rest "k8s.io/client-go/rest"
	flowcontrol "k8s.io/client-go/util/flowcontrol"
)

type Interface interface {
	Discovery() discovery.DiscoveryInterface
	BespinV1() bespinv1.BespinV1Interface
	// Deprecated: please explicitly pick a version if possible.
	Bespin() bespinv1.BespinV1Interface
	NomosV1() nomosv1.NomosV1Interface
	// Deprecated: please explicitly pick a version if possible.
	Nomos() nomosv1.NomosV1Interface
	NomosV1alpha1() nomosv1alpha1.NomosV1alpha1Interface
}

// Clientset contains the clients for groups. Each group has exactly one
// version included in a Clientset.
type Clientset struct {
	*discovery.DiscoveryClient
	bespinV1      *bespinv1.BespinV1Client
	nomosV1       *nomosv1.NomosV1Client
	nomosV1alpha1 *nomosv1alpha1.NomosV1alpha1Client
}

// BespinV1 retrieves the BespinV1Client
func (c *Clientset) BespinV1() bespinv1.BespinV1Interface {
	return c.bespinV1
}

// Deprecated: Bespin retrieves the default version of BespinClient.
// Please explicitly pick a version.
func (c *Clientset) Bespin() bespinv1.BespinV1Interface {
	return c.bespinV1
}

// NomosV1 retrieves the NomosV1Client
func (c *Clientset) NomosV1() nomosv1.NomosV1Interface {
	return c.nomosV1
}

// Deprecated: Nomos retrieves the default version of NomosClient.
// Please explicitly pick a version.
func (c *Clientset) Nomos() nomosv1.NomosV1Interface {
	return c.nomosV1
}

// NomosV1alpha1 retrieves the NomosV1alpha1Client
func (c *Clientset) NomosV1alpha1() nomosv1alpha1.NomosV1alpha1Interface {
	return c.nomosV1alpha1
}

// Discovery retrieves the DiscoveryClient
func (c *Clientset) Discovery() discovery.DiscoveryInterface {
	if c == nil {
		return nil
	}
	return c.DiscoveryClient
}

// NewForConfig creates a new Clientset for the given config.
func NewForConfig(c *rest.Config) (*Clientset, error) {
	configShallowCopy := *c
	if configShallowCopy.RateLimiter == nil && configShallowCopy.QPS > 0 {
		configShallowCopy.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(configShallowCopy.QPS, configShallowCopy.Burst)
	}
	var cs Clientset
	var err error
	cs.bespinV1, err = bespinv1.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	cs.nomosV1, err = nomosv1.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	cs.nomosV1alpha1, err = nomosv1alpha1.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}

	cs.DiscoveryClient, err = discovery.NewDiscoveryClientForConfig(&configShallowCopy)
	if err != nil {
		glog.Errorf("failed to create the DiscoveryClient: %v", err)
		return nil, err
	}
	return &cs, nil
}

// NewForConfigOrDie creates a new Clientset for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *Clientset {
	var cs Clientset
	cs.bespinV1 = bespinv1.NewForConfigOrDie(c)
	cs.nomosV1 = nomosv1.NewForConfigOrDie(c)
	cs.nomosV1alpha1 = nomosv1alpha1.NewForConfigOrDie(c)

	cs.DiscoveryClient = discovery.NewDiscoveryClientForConfigOrDie(c)
	return &cs
}

// New creates a new Clientset for the given RESTClient.
func New(c rest.Interface) *Clientset {
	var cs Clientset
	cs.bespinV1 = bespinv1.New(c)
	cs.nomosV1 = nomosv1.New(c)
	cs.nomosV1alpha1 = nomosv1alpha1.New(c)

	cs.DiscoveryClient = discovery.NewDiscoveryClient(c)
	return &cs
}
