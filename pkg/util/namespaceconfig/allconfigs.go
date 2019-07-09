package namespaceconfig

import (
	"time"

	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/kinds"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// AllConfigs holds things that Importer wants to sync. It is only used in-process, not written
// directly as a Kubernetes resource.
type AllConfigs struct {
	// Map of names to NamespaceConfigs.
	NamespaceConfigs map[string]v1.NamespaceConfig
	// Singleton config for non-CRD cluster-scoped resources.
	ClusterConfig *v1.ClusterConfig
	// Config with declared state for CRDs.
	CRDClusterConfig *v1.ClusterConfig
	// Map of names to Syncs.
	Syncs map[string]v1.Sync
	// ImportToken is the git hash of the repo when parsing began.
	ImportToken string
	// LoadTime is when parsing began.
	LoadTime metav1.Time
}

// NewAllConfigs initializes a default empty AllConfigs.
func NewAllConfigs(importToken string, t time.Time) *AllConfigs {
	loadTime := metav1.NewTime(t)
	return &AllConfigs{
		NamespaceConfigs: map[string]v1.NamespaceConfig{},
		ClusterConfig:    v1.NewClusterConfig(importToken, loadTime),
		CRDClusterConfig: v1.NewCRDClusterConfig(importToken, loadTime),
		Syncs:            map[string]v1.Sync{},
		ImportToken:      importToken,
		LoadTime:         loadTime,
	}
}

// AddClusterResource adds a cluster-scoped resource to the AllConfigs.
func (c *AllConfigs) AddClusterResource(o runtime.Object) {
	if o.GetObjectKind().GroupVersionKind() == kinds.CustomResourceDefinition() {
		// CRDs end up in their own ClusterConfig.
		c.CRDClusterConfig.AddResource(o)
	} else {
		c.ClusterConfig.AddResource(o)
	}
}

// AddNamespaceConfig adds a Namespace node to the AllConfigs.
func (c *AllConfigs) AddNamespaceConfig(name string, annotations map[string]string, labels map[string]string) {
	//TODO(b/137213356): What should we do with duplicate Namespaces?
	var resources []v1.GenericResources
	ns, found := c.NamespaceConfigs[name]
	if found {
		resources = ns.Spec.Resources
	}
	ns = *v1.NewNamespaceConfig(name, annotations, labels, c.ImportToken, c.LoadTime)
	ns.Spec.Resources = resources
	c.NamespaceConfigs[name] = ns
}

// AddNamespaceResource adds an object to a Namespace node, instantiating a default Namespace if
// none exists.
func (c *AllConfigs) AddNamespaceResource(namespace string, o runtime.Object) {
	ns, found := c.NamespaceConfigs[namespace]
	if !found {
		// TODO(b/137213356): What should we do when a Namespace doesn't exist?
		c.AddNamespaceConfig(namespace, nil, nil)
		ns = c.NamespaceConfigs[namespace]
	}
	ns.AddResource(o)
	c.NamespaceConfigs[namespace] = ns
}

// AddSync adds a sync to the AllConfigs, adding the required SyncFinalizer.
func (c *AllConfigs) AddSync(sync v1.Sync) {
	sync.SetFinalizers(append(sync.GetFinalizers(), v1.SyncFinalizer))
	c.Syncs[sync.Name] = sync
}
