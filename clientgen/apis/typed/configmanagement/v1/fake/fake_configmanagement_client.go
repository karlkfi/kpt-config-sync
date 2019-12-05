// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1 "github.com/google/nomos/clientgen/apis/typed/configmanagement/v1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeConfigmanagementV1 struct {
	*testing.Fake
}

func (c *FakeConfigmanagementV1) ClusterConfigs() v1.ClusterConfigInterface {
	return &FakeClusterConfigs{c}
}

func (c *FakeConfigmanagementV1) ClusterSelectors() v1.ClusterSelectorInterface {
	return &FakeClusterSelectors{c}
}

func (c *FakeConfigmanagementV1) HierarchyConfigs() v1.HierarchyConfigInterface {
	return &FakeHierarchyConfigs{c}
}

func (c *FakeConfigmanagementV1) NamespaceConfigs() v1.NamespaceConfigInterface {
	return &FakeNamespaceConfigs{c}
}

func (c *FakeConfigmanagementV1) NamespaceSelectors() v1.NamespaceSelectorInterface {
	return &FakeNamespaceSelectors{c}
}

func (c *FakeConfigmanagementV1) Repos() v1.RepoInterface {
	return &FakeRepos{c}
}

func (c *FakeConfigmanagementV1) Syncs() v1.SyncInterface {
	return &FakeSyncs{c}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeConfigmanagementV1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
