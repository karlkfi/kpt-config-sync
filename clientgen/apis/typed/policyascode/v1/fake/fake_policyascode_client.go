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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1 "github.com/google/nomos/clientgen/apis/typed/policyascode/v1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeBespinV1 struct {
	*testing.Fake
}

func (c *FakeBespinV1) Folders() v1.FolderInterface {
	return &FakeFolders{c}
}

func (c *FakeBespinV1) IAMPolicies(namespace string) v1.IAMPolicyInterface {
	return &FakeIAMPolicies{c, namespace}
}

func (c *FakeBespinV1) Organizations() v1.OrganizationInterface {
	return &FakeOrganizations{c}
}

func (c *FakeBespinV1) OrganizationPolicies(namespace string) v1.OrganizationPolicyInterface {
	return &FakeOrganizationPolicies{c, namespace}
}

func (c *FakeBespinV1) Projects(namespace string) v1.ProjectInterface {
	return &FakeProjects{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeBespinV1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
