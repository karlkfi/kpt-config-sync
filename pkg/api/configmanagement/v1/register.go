package v1

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: configmanagement.GroupName, Version: "v1"}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

var (
	// SchemeBuilder is the scheme builder for types in this package
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	// AddToScheme adds the types in this package ot a scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&NamespaceConfig{},
		&NamespaceConfigList{},
		&ClusterConfig{},
		&ClusterConfigList{},
		&ClusterSelector{},
		&NamespaceSelector{},
		&NamespaceSelectorList{},
		&Repo{},
		&RepoList{},
		&Sync{},
		&SyncList{},
		&HierarchyConfig{},
		&HierarchyConfigList{},
		&RepoSync{},
		&RepoSyncList{},
		&RootSync{},
		&RootSyncList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
