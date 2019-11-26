package testoutput

import (
	"path"
	"strings"
	"testing"

	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	visitortesting "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	fstesting "github.com/google/nomos/pkg/importer/filesystem/testing"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/util/discovery"
	"github.com/google/nomos/pkg/util/namespaceconfig"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func NewAllConfigs(t *testing.T, fileObjects ...ast.FileObject) *namespaceconfig.AllConfigs {
	result, errs := namespaceconfig.NewAllConfigs(visitortesting.ImportToken, metav1.Time{}, testScoper(t), fileObjects)
	if errs != nil {
		t.Fatal(errs)
	}
	return result
}

func testScoper(t *testing.T) discovery.Scoper {
	result, err := discovery.NewAPIInfo(fstesting.TestAPIResourceList(fstesting.TestDynamicResources()))
	if err != nil {
		t.Fatal(err)
	}
	return result
}

// ClusterConfig generates a valid ClusterConfig to be put in AllConfigs given the set of hydrated
// cluster-scoped runtime.Objects.
func ClusterConfig(objects ...core.Object) *v1.ClusterConfig {
	config := fake.ClusterConfigObject()
	config.Spec.Token = visitortesting.ImportToken
	for _, o := range objects {
		config.AddResource(o)
	}
	return config
}

// CRDClusterConfig generates a valid ClusterConfig which holds the list of CRDs in the repo.
func CRDClusterConfig(objects ...core.Object) *v1.ClusterConfig {
	config := fake.CRDClusterConfigObject()
	config.Spec.Token = visitortesting.ImportToken
	for _, o := range objects {
		config.AddResource(o)
	}
	return config
}

// NamespaceConfig generates a valid NamespaceConfig to be put in AllConfigs given the set of
// hydrated runtime.Objects for that Namespace.
func NamespaceConfig(clusterName, dir string, opt core.MetaMutator, objects ...core.Object) v1.NamespaceConfig {
	config := fake.NamespaceConfigObject(fake.NamespaceConfigMeta(Source(path.Join(dir, "namespace.yaml"))))
	config.Spec.Token = visitortesting.ImportToken
	if clusterName != "" {
		InCluster(clusterName)(config)
	}
	config.Name = cmpath.FromSlash(dir).Base()
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
