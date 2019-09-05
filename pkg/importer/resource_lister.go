package importer

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
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
func (l ResourceLister) List(apiResource metav1.APIResource) ([]ast.FileObject, status.MultiError) {
	if !listable(apiResource) {
		return nil, nil
	}

	gvr := schema.GroupVersionResource{
		Group:    apiResource.Group,
		Version:  apiResource.Version,
		Resource: apiResource.Name,
	}

	var items []unstructured.Unstructured
	for ok, token := true, ""; ok; ok = token != "" {
		// The token empty string gets the first page of 500, and we always want to request it.
		// We know we are at the last page when GetContinue() returns empty string.
		resources, err := l.resourcer.Resource(gvr).List(metav1.ListOptions{
			Continue: token,
		})
		if err != nil {
			return nil, status.APIServerWrapf(err, "unable to read %q resources", gvr.String())
		}
		items = append(items, resources.Items...)
		token = resources.GetContinue()
	}

	var result []ast.FileObject
	for _, r := range items {
		o := ast.ParseFileObject(r.DeepCopyObject())
		result = append(result, *o)
	}
	return result, nil
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
