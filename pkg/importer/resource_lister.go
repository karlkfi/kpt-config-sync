package importer

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Resourcer returns a Lister for given GroupVersionResource.
type Resourcer interface {
	// Resource mimics k8s.io/client-go/dynamic.Interface.Resource().
	Resource(resource schema.GroupVersionResource) Lister
}

// Lister returns an UnstructuredList of resources.
type Lister interface {
	// List mimics k8s.io/client-go/dynamic.ResourceInterface.List().
	List(opts metav1.ListOptions) (*unstructured.UnstructuredList, error)
}

// ResourceLister lists resources on a cluster.
type ResourceLister struct {
	resourcer Resourcer
}

// NewResourceLister initializes a ResourceLister from a Resourcer.
func NewResourceLister(resourcer Resourcer) ResourceLister {
	return ResourceLister{resourcer: resourcer}
}

// List returns all resources on the cluster of a given APIResource. If the APIResource is not
// listable, silently returns the empty list. Returns an error and the empty list if any were
// encountered listing the APIResource.
func (l ResourceLister) List(apiResource metav1.APIResource, errs ErrorAdder) []ast.FileObject {
	if !listable(apiResource) {
		return nil
	}

	gvr := schema.GroupVersionResource{
		Group:    apiResource.Group,
		Version:  apiResource.Version,
		Resource: apiResource.Name,
	}

	resources, err := l.resourcer.Resource(gvr).List(metav1.ListOptions{})
	errs.Add(status.APIServerWrapf(err, "unable to read %q resources", gvr.String()))
	if err != nil {
		return nil
	}
	// TODO(b/126702932): Check for resources.GetContinue value since there could be >500 resources.
	if resources.GetContinue() != "" {
		errs.Add(status.APIServerWrapf(err, "more than 500 %q resources; only exporting first 500", gvr.String()))
	}

	var result []ast.FileObject
	for _, r := range resources.Items {
		o := ast.FileObject{Object: r.DeepCopyObject()}
		result = append(result, o)
	}
	return result
}

// listable returns true if it is valid to use the "list" verb on the APIResource.
func listable(apiResource metav1.APIResource) bool {
	for _, verb := range apiResource.Verbs {
		if verb == "list" {
			return true
		}
	}
	return false
}
