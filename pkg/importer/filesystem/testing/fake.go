// Package testing contains fake implementation of the API discovery mechanisms,
// seeded with the types used in Nomos.  Use NewTestClientGetter first to create
// a new instance and work from there.
package testing

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/status"
	utildiscovery "github.com/google/nomos/pkg/util/discovery"
	openapi_v2 "github.com/googleapis/gnostic/OpenAPIv2"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
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
					{Name: "customresourcedefinitions", Namespaced: false, Kind: "CustomResourceDefinition"},
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
					{Name: "customresourcedefinitions", Namespaced: false, Kind: "CustomResourceDefinition"},
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

// Scoper returns a utildiscovery.Scoper with resources commonly used in testing.
//
// This includes many core Kubernetes types, as well as the internal Nomos types.
// Feel free to add new types as necessary.
func Scoper(crds ...*v1beta1.CustomResourceDefinition) utildiscovery.Scoper {
	var gkss []utildiscovery.GroupKindScope
	coreScopes := scopedKinds(corev1.GroupName, map[string]utildiscovery.ObjectScope{
		"Pod":                   utildiscovery.NamespaceScope,
		"Service":               utildiscovery.NamespaceScope,
		"ReplicationController": utildiscovery.NamespaceScope,
		"ComponentStatus":       utildiscovery.ClusterScope,
		"Node":                  utildiscovery.ClusterScope,
		"Secret":                utildiscovery.NamespaceScope,
		"ConfigMap":             utildiscovery.NamespaceScope,
		"Namespace":             utildiscovery.ClusterScope,
		"ResourceQuota":         utildiscovery.NamespaceScope,
	})
	gkss = append(gkss, coreScopes...)

	apiExtensionsScopes := scopedKinds(apiextensionsv1beta1.GroupName, map[string]utildiscovery.ObjectScope{
		"CustomResourceDefinition": utildiscovery.ClusterScope,
	})
	gkss = append(gkss, apiExtensionsScopes...)

	policyScopes := scopedKinds(policyv1beta1.GroupName, map[string]utildiscovery.ObjectScope{
		"PodSecurityPolicy": utildiscovery.ClusterScope,
	})
	gkss = append(gkss, policyScopes...)

	appsScopes := scopedKinds(appsv1.GroupName, map[string]utildiscovery.ObjectScope{
		"Deployment": utildiscovery.NamespaceScope,
		"ReplicaSet": utildiscovery.NamespaceScope,
	})
	gkss = append(gkss, appsScopes...)

	autoscalingScopes := scopedKinds(autoscalingv1.GroupName, map[string]utildiscovery.ObjectScope{
		"HorizontalPodAutoscaler": utildiscovery.NamespaceScope,
	})
	gkss = append(gkss, autoscalingScopes...)

	storageScopes := scopedKinds(storagev1beta1.GroupName, map[string]utildiscovery.ObjectScope{
		"StorageClass": utildiscovery.ClusterScope,
	})
	gkss = append(gkss, storageScopes...)

	rbacScopes := scopedKinds(rbacv1.GroupName, map[string]utildiscovery.ObjectScope{
		"Role":               utildiscovery.NamespaceScope,
		"RoleBinding":        utildiscovery.NamespaceScope,
		"ClusterRole":        utildiscovery.ClusterScope,
		"ClusterRoleBinding": utildiscovery.ClusterScope,
	})
	gkss = append(gkss, rbacScopes...)

	nomosScopes := scopedKinds(configmanagement.GroupName, map[string]utildiscovery.ObjectScope{
		"ClusterSelector":   utildiscovery.ClusterScope,
		"NamespaceSelector": utildiscovery.ClusterScope,
		"Repo":              utildiscovery.ClusterScope,
		"Sync":              utildiscovery.ClusterScope,
		"HierarchyConfig":   utildiscovery.ClusterScope,
		"NamespaceConfig":   utildiscovery.ClusterScope,
	})
	gkss = append(gkss, nomosScopes...)

	gkss = append(gkss, utildiscovery.ScopesFromCRDs(crds)...)

	result := utildiscovery.Scoper{}

	for _, gks := range gkss {
		result[gks.GroupKind] = gks.Scope
	}
	return result
}

func scopedKinds(group string, kindScope map[string]utildiscovery.ObjectScope) []utildiscovery.GroupKindScope {
	var result []utildiscovery.GroupKindScope
	for kind, scope := range kindScope {
		result = append(result, utildiscovery.GroupKindScope{
			GroupKind: schema.GroupKind{
				Group: group,
				Kind:  kind,
			},
			Scope: scope,
		})
	}
	return result
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
