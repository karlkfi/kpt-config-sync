/*
Copyright 2016 The Kubernetes Authors.

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

package testing

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/rest/fake"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
	api "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/kubectl"
	"k8s.io/kubernetes/pkg/kubectl/categories"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/kubectl/cmd/util/openapi"
	openapitesting "k8s.io/kubernetes/pkg/kubectl/cmd/util/openapi/testing"
	"k8s.io/kubernetes/pkg/kubectl/resource"
	"k8s.io/kubernetes/pkg/kubectl/validation"
	"k8s.io/kubernetes/pkg/printers"
)

type fakeCachedDiscoveryClient struct {
	discovery.DiscoveryInterface
}

func (d *fakeCachedDiscoveryClient) Fresh() bool {
	return true
}

func (d *fakeCachedDiscoveryClient) Invalidate() {
}

func (d *fakeCachedDiscoveryClient) ServerResources() ([]*metav1.APIResourceList, error) {
	return []*metav1.APIResourceList{}, nil
}

// TestFactory is a cmdutil.Factory that can be used in tests to avoid requiring talking
// to the API server for Discovery (need for RESTMapping) and downloading OpenAPI spec.
// Additional resources can be added to testDynamicTypes (e.g. kinds in nomos.dev group).
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

func NewTestFactory() *TestFactory {
	// specify an optionalClientConfig to explicitly use in testing
	// to avoid polluting an existing user config.
	config, configFile := defaultFakeClientConfig()
	return &TestFactory{
		Factory:        cmdutil.NewFactory(config),
		tempConfigFile: configFile,
	}
}

func (f *TestFactory) Cleanup() {
	if f.tempConfigFile == nil {
		return
	}

	os.Remove(f.tempConfigFile.Name())
}

func defaultFakeClientConfig() (clientcmd.ClientConfig, *os.File) {
	loadingRules, tmpFile, err := newDefaultFakeClientConfigLoadingRules()
	if err != nil {
		panic(fmt.Sprintf("unable to create a fake client config: %v", err))
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

func (f *TestFactory) CategoryExpander() categories.CategoryExpander {
	return categories.LegacyCategoryExpander
}

func (f *TestFactory) ClientConfig() (*restclient.Config, error) {
	return f.ClientConfigVal, nil
}

func (f *TestFactory) BareClientConfig() (*restclient.Config, error) {
	return f.ClientConfigVal, nil
}

func (f *TestFactory) ClientForMapping(mapping *meta.RESTMapping) (resource.RESTClient, error) {
	return f.Client, nil
}

func (f *TestFactory) UnstructuredClientForMapping(mapping *meta.RESTMapping) (resource.RESTClient, error) {
	if f.UnstructuredClientForMappingFunc != nil {
		return f.UnstructuredClientForMappingFunc(mapping)
	}
	return f.UnstructuredClient, nil
}

func (f *TestFactory) Describer(*meta.RESTMapping) (printers.Describer, error) {
	return f.DescriberVal, nil
}

func (f *TestFactory) Validator(validate bool) (validation.Schema, error) {
	return validation.NullSchema{}, nil
}

func (f *TestFactory) DefaultNamespace() (string, bool, error) {
	return f.Namespace, false, nil
}

func (f *TestFactory) OpenAPISchema() (openapi.Resources, error) {
	if f.OpenAPISchemaFunc != nil {
		return f.OpenAPISchemaFunc()
	}
	return openapitesting.EmptyResources{}, nil
}

func (f *TestFactory) Command(*cobra.Command, bool) string {
	return f.CommandVal
}

func (f *TestFactory) NewBuilder() *resource.Builder {
	mapper, typer := f.Object()

	return resource.NewBuilder(
		&resource.Mapper{
			RESTMapper:   mapper,
			ObjectTyper:  typer,
			ClientMapper: resource.ClientMapperFunc(f.ClientForMapping),
			Decoder:      cmdutil.InternalVersionDecoder(),
		},
		&resource.Mapper{
			RESTMapper:   mapper,
			ObjectTyper:  typer,
			ClientMapper: resource.ClientMapperFunc(f.UnstructuredClientForMapping),
			Decoder:      unstructured.UnstructuredJSONScheme,
		},
		f.CategoryExpander(),
	)
}

func (f *TestFactory) KubernetesClientSet() (*kubernetes.Clientset, error) {
	fakeClient := f.Client.(*fake.RESTClient)
	clientset := kubernetes.NewForConfigOrDie(f.ClientConfigVal)

	clientset.CoreV1().RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.AuthorizationV1().RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.AuthorizationV1beta1().RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.AuthorizationV1().RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.AuthorizationV1beta1().RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.AutoscalingV1().RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.AutoscalingV2beta1().RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.BatchV1().RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.BatchV2alpha1().RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.CertificatesV1beta1().RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.ExtensionsV1beta1().RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.RbacV1alpha1().RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.RbacV1beta1().RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.StorageV1().RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.StorageV1beta1().RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.AppsV1beta1().RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.AppsV1beta2().RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.PolicyV1beta1().RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.DiscoveryClient.RESTClient().(*restclient.RESTClient).Client = fakeClient.Client

	return clientset, nil
}

func (f *TestFactory) ClientSet() (internalclientset.Interface, error) {
	// Swap the HTTP client out of the REST client with the fake
	// version.
	fakeClient := f.Client.(*fake.RESTClient)
	clientset := internalclientset.NewForConfigOrDie(f.ClientConfigVal)
	clientset.Core().RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.Authentication().RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.Authorization().RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.Autoscaling().RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.Batch().RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.Certificates().RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.Extensions().RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.Rbac().RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.Storage().RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.Apps().RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.Policy().RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.DiscoveryClient.RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	return clientset, nil
}

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

func (f *TestFactory) DiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	fakeClient := f.Client.(*fake.RESTClient)
	discoveryClient := discovery.NewDiscoveryClientForConfigOrDie(f.ClientConfigVal)
	discoveryClient.RESTClient().(*restclient.RESTClient).Client = fakeClient.Client

	cacheDir := filepath.Join("", ".kube", "cache", "discovery")
	return cmdutil.NewCachedDiscoveryClient(discoveryClient, cacheDir, 10*time.Minute), nil
}

func (f *TestFactory) ClientSetForVersion(requiredVersion *schema.GroupVersion) (internalclientset.Interface, error) {
	return f.ClientSet()
}

func (f *TestFactory) Object() (meta.RESTMapper, runtime.ObjectTyper) {
	groupResources := testDynamicResources()
	mapper := discovery.NewRESTMapper(
		groupResources,
		meta.InterfacesForUnstructuredConversion(func(version schema.GroupVersion) (*meta.VersionInterfaces, error) {
			switch version {
			// provide typed objects for test objects here
			default:
				return legacyscheme.Registry.InterfacesFor(version)
			}
		}),
	)
	// for backwards compatibility with existing tests, allow rest mappings from the scheme to show up
	// TODO: make this opt-in?
	mapper = meta.FirstHitRESTMapper{
		MultiRESTMapper: meta.MultiRESTMapper{
			mapper,
			legacyscheme.Registry.RESTMapper(),
		},
	}

	// TODO: should probably be the external scheme
	typer := discovery.NewUnstructuredObjectTyper(groupResources, legacyscheme.Scheme)
	fakeDs := &fakeCachedDiscoveryClient{}
	expander := cmdutil.NewShortcutExpander(mapper, fakeDs)
	return expander, typer
}

func (f *TestFactory) LogsForObject(object, options runtime.Object, timeout time.Duration) (*restclient.Request, error) {
	c, err := f.ClientSet()
	if err != nil {
		panic(err)
	}

	switch t := object.(type) {
	case *api.Pod:
		opts, ok := options.(*api.PodLogOptions)
		if !ok {
			return nil, errors.New("provided options object is not a PodLogOptions")
		}
		return c.Core().Pods(f.Namespace).GetLogs(t.Name, opts), nil
	default:
		return nil, fmt.Errorf("cannot get the logs from %T", object)
	}
}

func testDynamicResources() []*discovery.APIGroupResources {
	return []*discovery.APIGroupResources{
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
					{Name: "clusterroles", Namespaced: false, Kind: "ClusterRole"},
				},
				"v1beta1": {
					{Name: "clusterrolebindings", Namespaced: false, Kind: "ClusterRoleBinding"},
				},
			},
		},
		{
			Group: metav1.APIGroup{
				Name: "nomos.dev",
				Versions: []metav1.GroupVersionForDiscovery{
					{Version: "v1"},
				},
				PreferredVersion: metav1.GroupVersionForDiscovery{Version: "v1"},
			},
			VersionedResources: map[string][]metav1.APIResource{
				"v1": {
					{Name: "namespaceselectors", Namespaced: true, Kind: "NamespaceSelector"},
				},
			},
		},
	}
}
