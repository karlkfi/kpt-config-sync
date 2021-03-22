package testoutput

import (
	"path"
	"strings"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const importToken = "abcde"

// ClusterConfig generates a valid ClusterConfig to be put in AllConfigs given the set of hydrated
// cluster-scoped client.Objects.
func ClusterConfig(objects ...client.Object) *v1.ClusterConfig {
	config := fake.ClusterConfigObject()
	config.Spec.Token = importToken
	for _, o := range objects {
		config.AddResource(o)
	}
	return config
}

// CRDClusterConfig generates a valid ClusterConfig which holds the list of CRDs in the repo.
func CRDClusterConfig(objects ...client.Object) *v1.ClusterConfig {
	config := fake.CRDClusterConfigObject()
	config.Spec.Token = importToken
	for _, o := range objects {
		config.AddResource(o)
	}
	return config
}

// NamespaceConfig generates a valid NamespaceConfig to be put in AllConfigs given the set of
// hydrated client.Objects for that Namespace.
func NamespaceConfig(clusterName, dir string, opt core.MetaMutator, objects ...client.Object) v1.NamespaceConfig {
	config := fake.NamespaceConfigObject(Source(path.Join(dir, "namespace.yaml")))
	config.Spec.Token = importToken
	if clusterName != "" {
		InCluster(clusterName)(config)
	}
	config.Name = cmpath.RelativeSlash(dir).Base()
	for _, o := range objects {
		o.SetNamespace(config.Name)
		config.AddResource(o)
	}
	if opt != nil {
		opt(config)
	}
	return *config
}

// NamespaceConfigs turns a list of NamespaceConfigs into the map AllConfigs requires.
func NamespaceConfigs(ncs ...v1.NamespaceConfig) map[string]v1.NamespaceConfig {
	result := map[string]v1.NamespaceConfig{}
	for _, nc := range ncs {
		result[nc.Name] = nc
	}
	return result
}

// Syncs generates the sync map to be put in AllConfigs.
func Syncs(gvks ...schema.GroupVersionKind) map[string]v1.Sync {
	result := map[string]v1.Sync{}
	for _, gvk := range gvks {
		result[GroupKind(gvk)] = *fake.SyncObject(gvk.GroupKind())
	}
	return result
}

// GroupKind factors out the two-line operation of getting the GroupKind string from a
// GroupVersionKind. The GroupKind.String() method has a pointer receiver, so
// gvk.GroupKind.String() is an error.
func GroupKind(gvk schema.GroupVersionKind) string {
	gk := gvk.GroupKind()
	return strings.ToLower(gk.String())
}
