// Package testing contains fake implementation of the API discovery mechanisms,
// seeded with the types used in Nomos.  Use NewTestClientGetter first to create
// a new instance and work from there.
package testing

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	utildiscovery "github.com/google/nomos/pkg/util/discovery"
	openapi_v2 "github.com/googleapis/gnostic/OpenAPIv2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/restmapper"
)

// NewTestClientGetter returns a new test RESTClientGetter that has mappings for test and provided resources.
func NewTestClientGetter(extraResources []*restmapper.APIGroupResources) utildiscovery.ClientGetter {
	return &fakeRESTClientGetter{newFakeCachedDiscoveryClient(testAPIResourceList(testDynamicResources(extraResources...)))}
}

// fakeRESTClientGetter implements utildiscovery.ClientGetter.
type fakeRESTClientGetter struct {
	DiscoveryClient discovery.CachedDiscoveryInterface
}

// ToDiscoveryClient returns discovery client
func (g *fakeRESTClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	return g.DiscoveryClient, nil
}

// fakeCachedDiscoveryClient is a DiscoveryClient with stubbed API Resources.
type fakeCachedDiscoveryClient struct {
	discovery.DiscoveryInterface
	APIGroupResources []*metav1.APIResourceList
}

// newFakeCachedDiscoveryClient returns a DiscoveryClient with stubbed API Resources.
func newFakeCachedDiscoveryClient(res []*metav1.APIResourceList) discovery.CachedDiscoveryInterface {
	return &fakeCachedDiscoveryClient{APIGroupResources: res}
}

// OpenAPISchema implements DiscoveryClient.
func (d *fakeCachedDiscoveryClient) OpenAPISchema() (*openapi_v2.Document, error) {
	return nil, nil
}

// Fresh always returns that the client is fresh.
func (d *fakeCachedDiscoveryClient) Fresh() bool {
	return true
}

// Invalidate is a no-op for the fake.
func (d *fakeCachedDiscoveryClient) Invalidate() {
}

// ServerResources returns the stubbed list of available resources.
func (d *fakeCachedDiscoveryClient) ServerResources() ([]*metav1.APIResourceList, error) {
	return d.APIGroupResources, nil
}

// ServerResourcesForGroupVersion returns the stubbed list of available resources in a given groupVersion.
func (d *fakeCachedDiscoveryClient) ServerResourcesForGroupVersion(groupVersion string) (*metav1.APIResourceList, error) {
	for _, list := range d.APIGroupResources {
		if list.GroupVersion == groupVersion {
			return list, nil
		}
	}
	return nil, status.InternalErrorf("%T wasn't given any %s resources", d, groupVersion)
}

// testAPIResourceList returns the API ResourceList as would be returned by the DiscoveryClient ServerResources
// call which represents resources that are returned by the API server during discovery.
func testAPIResourceList(rs []*restmapper.APIGroupResources) []*metav1.APIResourceList {
	var apiResources []*metav1.APIResourceList
	for _, item := range rs {
		for version, resources := range item.VersionedResources {
			apiResources = append(apiResources, &metav1.APIResourceList{
				TypeMeta: metav1.TypeMeta{
					APIVersion: metav1.SchemeGroupVersion.String(),
					Kind:       "APIResourceList",
				},
				GroupVersion: schema.GroupVersion{Group: item.Group.Name, Version: version}.String(),
				APIResources: resources,
			})
		}
	}
	return apiResources
}

func testK8SResources() []*restmapper.APIGroupResources {
	return []*restmapper.APIGroupResources{
		{
			Group: metav1.APIGroup{
				Versions: []metav1.GroupVersionForDiscovery{
					{Version: "v1"},
				},
				PreferredVersion: metav1.GroupVersionForDiscovery{Version: "v1"},
			},
			VersionedResources: map[string][]metav1.APIResource{
				"v1": {
					{Name: "pods", Namespaced: true, Kind: "Pod"},
					{Name: "services", Namespaced: true, Kind: "Service"},
					{Name: "replicationcontrollers", Namespaced: true, Kind: "ReplicationController"},
					{Name: "componentstatuses", Namespaced: false, Kind: "ComponentStatus"},
					{Name: "nodes", Namespaced: false, Kind: "Node"},
					{Name: "secrets", Namespaced: true, Kind: "Secret"},
					{Name: "configmaps", Namespaced: true, Kind: "ConfigMap"},
					{Name: "namespaces", Namespaced: false, Kind: "Namespace"},
					{Name: "resourcequotas", Namespaced: true, Kind: "ResourceQuota"},
				},
			},
		},
		{
			Group: metav1.APIGroup{
				Name: "apiextensions.k8s.io",
				Versions: []metav1.GroupVersionForDiscovery{
					{Version: "v1beta1"},
				},
				PreferredVersion: metav1.GroupVersionForDiscovery{Version: "v1beta1"},
			},
			VersionedResources: map[string][]metav1.APIResource{
				"v1beta1": {
					{Name: "customresourcedefinitions", Namespaced: false, Kind: kinds.CustomResourceDefinitionKind},
				},
			},
		},
		{
			Group: metav1.APIGroup{
				Name: "extensions",
				Versions: []metav1.GroupVersionForDiscovery{
					{Version: "v1beta1"},
				},
				PreferredVersion: metav1.GroupVersionForDiscovery{Version: "v1beta1"},
			},
			VersionedResources: map[string][]metav1.APIResource{
				"v1beta1": {
					{Name: "customresourcedefinitions", Namespaced: false, Kind: kinds.CustomResourceDefinitionKind},
				},
			},
		},
		{
			Group: metav1.APIGroup{
				Name: "policy",
				Versions: []metav1.GroupVersionForDiscovery{
					{Version: "v1beta1"},
				},
				PreferredVersion: metav1.GroupVersionForDiscovery{Version: "v1beta1"},
			},
			VersionedResources: map[string][]metav1.APIResource{
				"v1beta1": {
					{Name: "podsecuritypolicyies", Namespaced: false, Kind: "PodSecurityPolicy"},
				},
			},
		},
		{
			Group: metav1.APIGroup{
				Name: "apps",
				Versions: []metav1.GroupVersionForDiscovery{
					{Version: "v1beta2"},
				},
				PreferredVersion: metav1.GroupVersionForDiscovery{Version: "v1beta2"},
			},
			VersionedResources: map[string][]metav1.APIResource{
				"v1beta2": {
					{Name: "deployments", Namespaced: true, Kind: "Deployment"},
					{Name: "replicasets", Namespaced: true, Kind: "ReplicaSet"},
				},
			},
		},
		{
			Group: metav1.APIGroup{
				Name: "apps",
				Versions: []metav1.GroupVersionForDiscovery{
					{Version: "v1beta1"},
					{Version: "v1beta2"},
					{Version: "v1"},
				},
				PreferredVersion: metav1.GroupVersionForDiscovery{Version: "v1"},
			},
			VersionedResources: map[string][]metav1.APIResource{
				"v1beta1": {
					{Name: "deployments", Namespaced: true, Kind: "Deployment"},
					{Name: "replicasets", Namespaced: true, Kind: "ReplicaSet"},
				},
				"v1beta2": {
					{Name: "deployments", Namespaced: true, Kind: "Deployment"},
				},
				"v1": {
					{Name: "deployments", Namespaced: true, Kind: "Deployment"},
					{Name: "replicasets", Namespaced: true, Kind: "ReplicaSet"},
				},
			},
		},
		{
			Group: metav1.APIGroup{
				Name: "autoscaling",
				Versions: []metav1.GroupVersionForDiscovery{
					{Version: "v1"},
					{Version: "v2beta1"},
				},
				PreferredVersion: metav1.GroupVersionForDiscovery{Version: "v2beta1"},
			},
			VersionedResources: map[string][]metav1.APIResource{
				"v1": {
					{Name: "horizontalpodautoscalers", Namespaced: true, Kind: "HorizontalPodAutoscaler"},
				},
				"v2beta1": {
					{Name: "horizontalpodautoscalers", Namespaced: true, Kind: "HorizontalPodAutoscaler"},
				},
			},
		},
		{
			Group: metav1.APIGroup{
				Name: "storage.k8s.io",
				Versions: []metav1.GroupVersionForDiscovery{
					{Version: "v1beta1"},
					{Version: "v0"},
				},
				PreferredVersion: metav1.GroupVersionForDiscovery{Version: "v1beta1"},
			},
			VersionedResources: map[string][]metav1.APIResource{
				"v1beta1": {
					{Name: "storageclasses", Namespaced: false, Kind: "StorageClass"},
				},
				// bogus version of a known group/version/resource to make sure kubectl falls back to generic object mode
				"v0": {
					{Name: "storageclasses", Namespaced: false, Kind: "StorageClass"},
				},
			},
		},
		{
			Group: metav1.APIGroup{
				Name: "rbac.authorization.k8s.io",
				Versions: []metav1.GroupVersionForDiscovery{
					{Version: "v1beta1"},
					{Version: "v1"},
				},
				PreferredVersion: metav1.GroupVersionForDiscovery{Version: "v1"},
			},
			VersionedResources: map[string][]metav1.APIResource{
				"v1": {
					{Name: "roles", Namespaced: true, Kind: "Role"},
					{Name: "rolebindings", Namespaced: true, Kind: "RoleBinding"},
					{Name: "clusterroles", Namespaced: false, Kind: "ClusterRole"},
					{Name: "clusterrolebindings", Namespaced: false, Kind: "ClusterRoleBinding"},
				},
				"v1beta1": {
					{Name: "clusterrolebindings", Namespaced: false, Kind: "ClusterRoleBinding"},
				},
			},
		},
	}
}

// testDynamicResources returns API Resources for both standard K8S resources
// and Nomos resources.
func testDynamicResources(extraResources ...*restmapper.APIGroupResources) []*restmapper.APIGroupResources {
	r := testK8SResources()
	r = append(r, []*restmapper.APIGroupResources{
		{
			Group: metav1.APIGroup{
				Name: configmanagement.GroupName,
				Versions: []metav1.GroupVersionForDiscovery{
					{Version: "v1alpha1"},
					{Version: "v1"},
				},
				PreferredVersion: metav1.GroupVersionForDiscovery{Version: "v1alpha1"},
			},
			VersionedResources: map[string][]metav1.APIResource{
				"v1alpha1": {
					{Name: "clusterselectors", Namespaced: false, Kind: configmanagement.ClusterSelectorKind},
					{Name: "namespaceselectors", Namespaced: false, Kind: configmanagement.NamespaceSelectorKind},
					{Name: "repos", Namespaced: false, Kind: configmanagement.RepoKind},
					{Name: "syncs", Namespaced: false, Kind: configmanagement.SyncKind},
					{Name: "hierarchyconfigs", Namespaced: false, Kind: configmanagement.HierarchyConfigKind},
					{Name: "namespaceconfigs", Namespaced: false, Kind: configmanagement.NamespaceConfigKind},
				},
				"v1": {
					{Name: "clusterselectors", Namespaced: false, Kind: configmanagement.ClusterSelectorKind},
					{Name: "namespaceselectors", Namespaced: false, Kind: configmanagement.NamespaceSelectorKind},
					{Name: "repos", Namespaced: false, Kind: configmanagement.RepoKind},
					{Name: "syncs", Namespaced: false, Kind: configmanagement.SyncKind},
					{Name: "hierarchyconfigs", Namespaced: false, Kind: configmanagement.HierarchyConfigKind},
					{Name: "namespaceconfigs", Namespaced: false, Kind: configmanagement.NamespaceConfigKind},
				},
			},
		},
		{
			Group: metav1.APIGroup{
				Name: "clusterregistry.k8s.io",
				Versions: []metav1.GroupVersionForDiscovery{
					{Version: "v1alpha1"},
				},
				PreferredVersion: metav1.GroupVersionForDiscovery{Version: "v1alpha1"},
			},
			VersionedResources: map[string][]metav1.APIResource{
				"v1alpha1": {
					{Name: "clusters", Namespaced: false, Kind: "Cluster"},
				},
			},
		},
	}...,
	)
	r = append(r, extraResources...)
	return r
}
