package namespaceconfig

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/transform"
	"github.com/google/nomos/pkg/importer/analyzer/validation"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/discovery"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
}

// NewAllConfigs initializes a default empty AllConfigs.
func NewAllConfigs(importToken string, loadTime metav1.Time, scoper discovery.Scoper, fileObjects []ast.FileObject) (*AllConfigs, status.MultiError) {
	result := &AllConfigs{
		NamespaceConfigs: map[string]v1.NamespaceConfig{},
		ClusterConfig:    v1.NewClusterConfig(importToken, loadTime),
		CRDClusterConfig: v1.NewCRDClusterConfig(importToken, loadTime),
		Syncs:            map[string]v1.Sync{},
	}

	var errs status.MultiError
	for _, f := range fileObjects {
		if transform.IsEphemeral(f.GroupVersionKind()) {
			// Do not materialize NamespaceSelectors.
			continue
		}

		if f.GroupVersionKind() == kinds.Namespace() {
			// Namespace is a snowflake.
			// This preserves the ordering behavior of kubectl apply -f. This means what is in the
			// alphabetically-last file wins.
			result.addNamespaceConfig(f.GetName(), importToken, loadTime, f.GetAnnotations(), f.GetLabels())
			continue
		}

		result.addSync(*v1.NewSync(f.GroupVersionKind().GroupKind()))
		switch scoper.GetScope(f.GroupVersionKind().GroupKind()) {
		case discovery.ClusterScope:
			result.addClusterResource(f.Object)
		case discovery.NamespaceScope:
			namespace := f.GetNamespace()
			if namespace == "" {
				// Empty string/non-declared metadata.namespace automatically maps to "default", so this
				// ensures we maintain these in a single NamespaceConfig entry.
				namespace = "default"
			}
			result.addNamespaceResource(namespace, importToken, loadTime, f.Object)
		case discovery.UnknownScope:
			spew.Dump(f.GroupVersionKind())
			errs = status.Append(errs, validation.UnknownObjectError(f))
		}
	}

	return result, errs
}

// addClusterResource adds a cluster-scoped resource to the AllConfigs.
func (c *AllConfigs) addClusterResource(o core.Object) {
	if o.GroupVersionKind() == kinds.CustomResourceDefinition() {
		// CRDs end up in their own ClusterConfig.
		c.CRDClusterConfig.AddResource(o)
	} else {
		c.ClusterConfig.AddResource(o)
	}
}

// addNamespaceConfig adds a Namespace node to the AllConfigs.
func (c *AllConfigs) addNamespaceConfig(name string, importToken string, loadTime metav1.Time, annotations map[string]string, labels map[string]string) {
	//TODO(b/137213356): What should we do with duplicate Namespaces?
	var resources []v1.GenericResources
	ns, found := c.NamespaceConfigs[name]
	if found {
		resources = ns.Spec.Resources
	}
	ns = *v1.NewNamespaceConfig(name, annotations, labels, importToken, loadTime)
	ns.Spec.Resources = resources
	c.NamespaceConfigs[name] = ns
}

// addNamespaceResource adds an object to a Namespace node, instantiating a default Namespace if
// none exists.
func (c *AllConfigs) addNamespaceResource(namespace string, importToken string, loadTime metav1.Time, o core.Object) {
	ns, found := c.NamespaceConfigs[namespace]
	if !found {
		// TODO(b/137213356): What should we do when a Namespace doesn't exist?
		c.addNamespaceConfig(namespace, importToken, loadTime, nil, nil)
		ns = c.NamespaceConfigs[namespace]
	}
	ns.AddResource(o)
	c.NamespaceConfigs[namespace] = ns
}

// addSync adds a sync to the AllConfigs, adding the required SyncFinalizer.
func (c *AllConfigs) addSync(sync v1.Sync) {
	sync.SetFinalizers(append(sync.GetFinalizers(), v1.SyncFinalizer))
	c.Syncs[sync.Name] = sync
}
