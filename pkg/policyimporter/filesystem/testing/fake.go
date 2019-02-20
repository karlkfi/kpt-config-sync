/*
Copyright 2018 The Nomos Authors

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

// Package testing contains fake implementation of the API discovery mechanisms,
// seeded with the types used in Nomos.  Use NewTestFactory first to create
// a new instance and work from there.
package testing

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/nomos/pkg/api/policyhierarchy"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	fakediscovery "k8s.io/client-go/discovery/fake"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/rest/fake"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/kubernetes/pkg/kubectl"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/kubectl/cmd/util/openapi"
	openapitesting "k8s.io/kubernetes/pkg/kubectl/cmd/util/openapi/testing"
	"k8s.io/kubernetes/pkg/kubectl/genericclioptions/resource"
	"k8s.io/kubernetes/pkg/kubectl/validation"
	"k8s.io/kubernetes/pkg/printers"
)

// FakeRESTClientGetter implements RESTClientGetter.
type FakeRESTClientGetter struct {
	Config          clientcmd.ClientConfig
	DiscoveryClient discovery.CachedDiscoveryInterface
	RestMapper      meta.RESTMapper
}

// ToRESTConfig returns restconfig
func (g *FakeRESTClientGetter) ToRESTConfig() (*restclient.Config, error) {
	return g.Config.ClientConfig()
}

// ToDiscoveryClient returns discovery client
func (g *FakeRESTClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	return g.DiscoveryClient, nil
}

// ToRESTMapper returns a restmapper
func (g *FakeRESTClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	return g.RestMapper, nil
}

// ToRawKubeConfigLoader return kubeconfig loader as-is
func (g *FakeRESTClientGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return g.Config
}

// FakeCachedDiscoveryClient is a DiscoveryClient with stubbed API Resources.
type FakeCachedDiscoveryClient struct {
	discovery.DiscoveryInterface
	APIGroupResources []*metav1.APIResourceList
}

// NewFakeCachedDiscoveryClient returns a DiscoveryClient with stubbed API Resources.
func NewFakeCachedDiscoveryClient(res []*metav1.APIResourceList) discovery.CachedDiscoveryInterface {
	return &FakeCachedDiscoveryClient{APIGroupResources: res}
}

// Fresh always returns that the client is fresh.
func (d *FakeCachedDiscoveryClient) Fresh() bool {
	return true
}

// Invalidate is a no-op for the fake.
func (d *FakeCachedDiscoveryClient) Invalidate() {
}

// ServerResources returns the stubbed list of available resources.
func (d *FakeCachedDiscoveryClient) ServerResources() ([]*metav1.APIResourceList, error) {
	return d.APIGroupResources, nil
}

// ServerResourcesForGroupVersion returns the stubbed list of available resources in a given groupVersion.
func (d *FakeCachedDiscoveryClient) ServerResourcesForGroupVersion(groupVersion string) (*metav1.APIResourceList, error) {
	for _, list := range d.APIGroupResources {
		if list.GroupVersion == groupVersion {
			return list, nil
		}
	}
	return nil, vet.InternalErrorf("%T wasn't given any %s resources", d, groupVersion)
}

// TestFactory is a cmdutil.Factory that can be used in tests to avoid requiring talking
// to the API server for Discovery (need for RESTMapping) and downloading OpenAPI spec.
// Additional resources can be added to TestDynamicTypes (e.g. kinds in nomos.dev group).
type TestFactory struct {
	cmdutil.Factory

	Client             kubectl.RESTClient
	UnstructuredClient kubectl.RESTClient
	DescriberVal       printers.Describer
	Namespace          string
	ClientConfigVal    *restclient.Config
	CommandVal         string

	tempConfigFile *os.File

	UnstructuredClientForMappingFunc func(mapping *meta.RESTMapping) (resource.RESTClient, error)
	OpenAPISchemaFunc                func() (openapi.Resources, error)
}

// NewTestFactory returns a new test factory.
func NewTestFactory(t *testing.T, extraResources ...*restmapper.APIGroupResources) *TestFactory {
	// specify an optionalClientConfig to explicitly use in testing
	// to avoid polluting an existing user Config.
	config, configFile := defaultFakeClientConfig(t)
	rConfig, _ := config.ClientConfig()
	return &TestFactory{
		Factory: cmdutil.NewFactory(
			&FakeRESTClientGetter{
				Config:          config,
				DiscoveryClient: NewFakeCachedDiscoveryClient(TestAPIResourceList(TestDynamicResources(extraResources...))),
			}),
		Client:          &fake.RESTClient{},
		tempConfigFile:  configFile,
		ClientConfigVal: rConfig,
	}
}

// Cleanup cleans up temporary files generated by the test run.
func (f *TestFactory) Cleanup() error {
	if f.tempConfigFile == nil {
		return nil
	}

	return os.Remove(f.tempConfigFile.Name())
}

func defaultFakeClientConfig(t *testing.T) (clientcmd.ClientConfig, *os.File) {
	loadingRules, tmpFile, err := newDefaultFakeClientConfigLoadingRules()
	if err != nil {
		t.Fatal(fmt.Sprintf("unable to create a fake client Config: %v", err))
	}

	overrides := &clientcmd.ConfigOverrides{ClusterDefaults: clientcmdapi.Cluster{Server: "http://localhost:8080"}}
	fallbackReader := bytes.NewBuffer([]byte{})

	clientConfig := clientcmd.NewInteractiveDeferredLoadingClientConfig(loadingRules, overrides, fallbackReader)
	return clientConfig, tmpFile
}

func newDefaultFakeClientConfigLoadingRules() (*clientcmd.ClientConfigLoadingRules, *os.File, error) {
	tmpFile, err := ioutil.TempFile("", "cmdtests_temp")
	if err != nil {
		return nil, nil, err
	}

	return &clientcmd.ClientConfigLoadingRules{
		Precedence:     []string{tmpFile.Name()},
		MigrationRules: map[string]string{},
	}, tmpFile, nil
}

// ClientForMapping returns the structured client for a given mapping.
func (f *TestFactory) ClientForMapping(mapping *meta.RESTMapping) (resource.RESTClient, error) {
	return f.Client, nil
}

// UnstructuredClientForMapping returns the unstructured client for a given mapping.
func (f *TestFactory) UnstructuredClientForMapping(mapping *meta.RESTMapping) (resource.RESTClient, error) {
	if f.UnstructuredClientForMappingFunc != nil {
		return f.UnstructuredClientForMappingFunc(mapping)
	}
	return f.UnstructuredClient, nil
}

// Validator returns a null validation schema.
func (f *TestFactory) Validator(validate bool) (validation.Schema, error) {
	return validation.NullSchema{}, nil
}

// OpenAPISchema returns the OpenAPI Schema.
func (f *TestFactory) OpenAPISchema() (openapi.Resources, error) {
	if f.OpenAPISchemaFunc != nil {
		return f.OpenAPISchemaFunc()
	}
	return openapitesting.EmptyResources{}, nil
}

// NewBuilder returns a new resource builder.
func (f *TestFactory) NewBuilder() *resource.Builder {
	fn := func(version schema.GroupVersion) (resource.RESTClient, error) {
		return f.ClientForMapping(nil)
	}
	mapper := f.RestMapper()
	dc := &fakediscovery.FakeDiscovery{}
	return resource.NewFakeBuilder(fn, mapper, restmapper.NewDiscoveryCategoryExpander(dc))
}

// RESTClient returns a rest client.
func (f *TestFactory) RESTClient() (*restclient.RESTClient, error) {
	// Swap out the HTTP client out of the client with the fake's version.
	fakeClient := f.Client.(*fake.RESTClient)
	restClient, err := restclient.RESTClientFor(f.ClientConfigVal)
	if err != nil {
		panic(err)
	}
	restClient.Client = fakeClient.Client
	return restClient, nil
}

// DiscoveryClient returns a discovery client.
func (f *TestFactory) DiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	fakeClient := f.Client.(*fake.RESTClient)
	discoveryClient := discovery.NewDiscoveryClientForConfigOrDie(f.ClientConfigVal)
	discoveryClient.RESTClient().(*restclient.RESTClient).Client = fakeClient.Client

	cacheDir := filepath.Join("", ".kube", "cache", "discovery")
	return discovery.NewCachedDiscoveryClientForConfig(f.ClientConfigVal, cacheDir, cacheDir, 10*time.Minute)
}

// RestMapper returns a RESTMapper.
func (f *TestFactory) RestMapper() meta.RESTMapper {
	return restmapper.NewDiscoveryRESTMapper(TestDynamicResources())
}

// TestAPIResourceList returns the API ResourceList as would be returned by the DiscoveryClient ServerResources
// call which represents resources that are returned by the API server during discovery.
func TestAPIResourceList(rs []*restmapper.APIGroupResources) []*metav1.APIResourceList {
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
					{Name: "namespacedtype", Namespaced: true, Kind: "NamespacedType"},
					{Name: "namespaces", Namespaced: false, Kind: "Namespace"},
					{Name: "resourcequotas", Namespaced: true, Kind: "ResourceQuota"},
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
					{Name: "deployments", Namespaced: true, Kind: "Deployment"},
					{Name: "replicasets", Namespaced: true, Kind: "ReplicaSet"},
					{Name: "podsecuritypolicyies", Namespaced: false, Kind: "PodSecurityPolicy"},
					{Name: "customresourcedefinitions", Namespaced: false, Kind: "CustomResourceDefinition"},
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

// TestDynamicResources returns API Resources for both standard K8S resources
// and Nomos resources.
func TestDynamicResources(extraResources ...*restmapper.APIGroupResources) []*restmapper.APIGroupResources {
	r := testK8SResources()
	r = append(r, []*restmapper.APIGroupResources{
		{
			Group: metav1.APIGroup{
				Name: policyhierarchy.GroupName,
				Versions: []metav1.GroupVersionForDiscovery{
					{Version: "v1alpha1"},
					{Version: "v1"},
				},
				PreferredVersion: metav1.GroupVersionForDiscovery{Version: "v1alpha1"},
			},
			VersionedResources: map[string][]metav1.APIResource{
				"v1alpha1": {
					{Name: "clusterselectors", Namespaced: false, Kind: "ClusterSelector"},
					{Name: "namespaceselectors", Namespaced: false, Kind: "NamespaceSelector"},
					{Name: "repos", Namespaced: false, Kind: "Repo"},
					{Name: "syncs", Namespaced: false, Kind: "Sync"},
					{Name: "hierarchyconfigs", Namespaced: false, Kind: "HierarchyConfig"},
					{Name: "policynodes", Namespaced: false, Kind: "PolicyNode"},
				},
				"v1": {
					{Name: "policynodes", Namespaced: false, Kind: "PolicyNode"},
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
		{
			Group: metav1.APIGroup{
				Name: "employees",
				Versions: []metav1.GroupVersionForDiscovery{
					{Version: "v1alpha1"},
				},
				PreferredVersion: metav1.GroupVersionForDiscovery{Version: "v1alpha1"},
			},
			VersionedResources: map[string][]metav1.APIResource{
				"v1alpha1": {
					{Name: "engineers", Namespaced: true, Kind: "Engineer"},
				},
			},
		},
	}...,
	)
	r = append(r, extraResources...)
	return r
}
