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

package fake

import (
	v1 "github.com/google/stolos/pkg/client/policyhierarchy/typed/k8us/v1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeK8usV1 struct {
	*testing.Fake
}

func (c *FakeK8usV1) PolicyNodes() v1.PolicyNodeInterface {
	return &FakePolicyNodes{c}
}

func (c *FakeK8usV1) StolosResourceQuotas(namespace string) v1.StolosResourceQuotaInterface {
	return &FakeStolosResourceQuotas{c, namespace}
}

func (c *FakeK8usV1) StolosRoleBindings(namespace string) v1.StolosRoleBindingInterface {
	return &FakeStolosRoleBindings{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeK8usV1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
