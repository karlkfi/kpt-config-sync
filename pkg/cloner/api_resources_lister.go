package cloner

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// APIResourcesLister mimics the smallest required part of discovery.ServerResourcesInterface which
// lists resources available to a Kubernetes cluster.
type APIResourcesLister interface {
	// ServerResources returns the supported API resources for all groups and versions.
	ServerResources() ([]*metav1.APIResourceList, error)
}

// ListResources calls the APIResourcesLister to retrieve the set of APIResources supported
// on the server, and then converts them to a list of GroupVersionResource. Only returns
// APIResources which support the "list" verb.
//
// Currently unused; will be used once `clone` is implemented.
func ListResources(lister APIResourcesLister) ([]metav1.APIResource, error) {
	apiResources, err := lister.ServerResources()
	if err != nil {
		return nil, errors.Wrapf(err, "unable to list supported API resources")
	}

	return flatten(apiResources)
}

// groupVersionParseErrors holds all errors encountered parsing GroupVersions.
// These should be rare to nonexistent in practice, only occurring if the cluster returns corrupted
// information.
type groupVersionParseErrors struct {
	errs []error
}

// Error implements vet.Error.
func (e *groupVersionParseErrors) Error() string {
	var errs []string
	for _, err := range e.errs {
		errs = append(errs, err.Error())
	}
	return fmt.Sprintf("errors parsing apiVersions:\n%s", strings.Join(errs, "\n"))
}

// add records a new error parsing an apiVersion string.
func (e *groupVersionParseErrors) add(err error) {
	if err == nil {
		return
	}
	e.errs = append(e.errs, err)
}

// err returns nil if there were no errors, otherwise a vet.Error.
func (e *groupVersionParseErrors) err() error {
	if len(e.errs) == 0 {
		return nil
	}
	return e
}

// flatten returns a list of the listable APIResources contained in the list of list of
// APIResources. If a given group/resource has multiple versions, returns the most recent.
func flatten(lists []*metav1.APIResourceList) ([]metav1.APIResource, error) {
	errs := groupVersionParseErrors{}
	grs := make(map[schema.GroupResource][]metav1.APIResource)

	for _, list := range lists {
		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			errs.add(err)
			// Encountering an error parsing this list doesn't prevent parsing the other lists.
			continue
		}

		for _, resource := range list.APIResources {
			// The server is not guaranteed to populate Group and Version, so this ensures it happens.
			resource.Group = gv.Group
			resource.Version = gv.Version
			gr := schema.GroupResource{
				Group:    gv.Group,
				Resource: resource.Name,
			}
			grs[gr] = append(grs[gr], resource)
		}
	}

	var result []metav1.APIResource
	for _, resources := range grs {
		sort.Slice(resources, func(i, j int) bool {
			return lessVersions(resources[i], resources[j])
		})
		result = append(result, resources[0])
	}
	return result, errs.err()
}
