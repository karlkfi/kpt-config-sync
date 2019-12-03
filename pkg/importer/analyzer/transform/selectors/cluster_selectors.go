package selectors

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

// state represents what we know about whether an object should be synced to the cluster
// based on the declared ClusterSelectors.
type state string

const (
	// active represents objects that should be synced to the cluster.
	active = state("active")
	// inactive represents objects that should NOT be synced to the cluster.
	inactive = state("inactive")
	// unknown is the value we use when we encounter a problem and can't determine
	// whether the object should be synced.
	unknown = state("unknown")
)

// ResolveClusterSelectors returns the list of objects which should be synced to cluster
// clusterName based on the ClusterSelectors.
//
// Rules:
// 1. A ClusterSelector is active if its LabelSelector selects Cluster clusterName.
//      Otherwise it is inactive.
// 2. If there is no Cluster declaration for ClusterName, all ClusterSelectors are inactive.
// 3. If an object's ClusterSelector is inactive, it is excluded.
// 4. If an object is in an excluded Namespace, it is excluded.
//
// Returns error(s) if:
// - an object references a ClusterSelector which does not exist,
// - a ClusterSelector is invalid or empty.
func ResolveClusterSelectors(clusterName string, objects []ast.FileObject) ([]ast.FileObject, status.MultiError) {
	// Get the Cluster matching "clusterName".
	clusters := FilterClusters(objects)
	cluster := getCluster(clusterName, clusters)

	// Get the active/inactive state for all declared ClusterSelectors.
	csStates, err := getClusterSelectorStates(cluster, objects)
	if err != nil {
		return nil, err
	}

	// Determine whether each Namespace is active.
	nsStates := getNamespaceStates(csStates, objects)

	// Discard objects and Namespaces with inactive ClusterSelectors.
	return resolveClusterSelectors(csStates, nsStates, objects)
}

// FilterClusters returns the list of Clusters in the passed array of FileObjects.
func FilterClusters(objects []ast.FileObject) []clusterregistry.Cluster {
	var clusters []clusterregistry.Cluster
	for _, object := range objects {
		if o, ok := object.Object.(*clusterregistry.Cluster); ok {
			clusters = append(clusters, *o)
		}
	}
	return clusters
}

// resolveClusterSelectors returns a list of FileObjects either referencing active
// ClusterSelectors, or that do not declare ClusterSelectors.
//
// Returns an Error if any passed objects reference undeclared ClusterSelectors.
func resolveClusterSelectors(csStates map[string]state, nsStates map[string]state, objects []ast.FileObject) ([]ast.FileObject, status.MultiError) {
	var result []ast.FileObject
	var errs status.MultiError
	for _, object := range objects {
		// Discard Clusters and ClusterSelectors as we don't need them anymore.
		if gvk := object.GroupVersionKind(); gvk == kinds.Cluster() || gvk == kinds.ClusterSelector() {
			continue
		}

		// Given the active/inactive states of every ClusterSelector and Namespace,
		// determine whether the object appears on the cluster.
		objState, err := objectState(csStates, nsStates, object)
		if err != nil {
			errs = status.Append(errs, err)
			continue
		}
		if objState == active {
			// The object is active on this cluster, so keep it.
			// This removes all inactive objects by omission.
			result = append(result, object)
		}
	}
	return result, errs
}

// objectState returns the active/inactive state for the object. This is determined by
//
// 1. If the object declares a ClusterSelector annotation that is inactive, the object is inactive.
// 2. If the object declares a metadata.namespace for a Namespace that is inactive, the object is inactive.
// 3. Otherwise, the object is active.
//
// Returns an error if the object references an undeclared ClusterSelector.
func objectState(csStates map[string]state, nsStates map[string]state, object ast.FileObject) (state, status.Error) {
	if nsState, nsDefined := nsStates[object.GetNamespace()]; nsDefined {
		// object is in an inactive Namespace, so it is inactive.
		if nsState == inactive {
			return inactive, nil
		}
	}

	annotation, hasAnnotation := object.GetAnnotations()[v1.ClusterSelectorAnnotationKey]
	if !hasAnnotation {
		// object has no annotation, so it is active.
		return active, nil
	}

	csState, csDefined := csStates[annotation]
	if !csDefined {
		// We require that all objects which declare the ClusterSelector annotation reference
		// a ClusterSelector that exists.
		return unknown, ObjectHasUnknownClusterSelector(object, annotation)
	}

	// object inherits the state of its ClusterSelector.
	return csState, nil
}

// getNamespaceStates returns a map from all defined Namespaces, and whether they are active or
// inactive on the cluster.
func getNamespaceStates(csStates map[string]state, objects []ast.FileObject) map[string]state {
	result := make(map[string]state)

	var errs status.MultiError
	for _, object := range objects {
		if object.GroupVersionKind() != kinds.Namespace() {
			// Ignore non-Namespaces
			continue
		}

		nsState, err := objectState(csStates, nil, object)
		if err != nil {
			errs = status.Append(errs, err)
			continue
		}
		result[object.GetName()] = nsState
	}

	return result
}

// getClusterSelectorStates returns the names of all active ClusterSelectors.
func getClusterSelectorStates(cluster *clusterregistry.Cluster, objects []ast.FileObject) (map[string]state, status.MultiError) {
	// ClusterSelectors may only select Clusters with definitions.
	selectors := filterClusterSelectors(objects)

	result := make(map[string]state)
	var errs status.MultiError
	for _, selector := range selectors {
		isSelected, err := selects(selector, cluster)
		if err != nil {
			errs = status.Append(errs, err)
			continue
		}

		if isSelected {
			result[selector.Name] = active
		} else {
			result[selector.Name] = inactive
		}
	}
	return result, errs
}

// getCluster returns the Cluster with name clusterName, or nil if it does not exist.
func getCluster(clusterName string, clusters []clusterregistry.Cluster) *clusterregistry.Cluster {
	for _, c := range clusters {
		if c.Name == clusterName {
			return &c
		}
	}
	return nil
}

// clusterSelectorFileObject is basically a FileObject that can only hold a ClusterSelector.
// This is a convenience struct that extends ClusterSelector to satisfy id.Resource,
// enabling us to generate good error messages about it.
type clusterSelectorFileObject struct {
	cmpath.Path
	*v1.ClusterSelector
}

// filterClusterSelectors returns the list of ClustersSelectors in the passed array of FileObjects.
func filterClusterSelectors(objects []ast.FileObject) []clusterSelectorFileObject {
	var clusterSelectors []clusterSelectorFileObject
	for _, object := range objects {
		if o, ok := object.Object.(*v1.ClusterSelector); ok {
			clusterSelectors = append(clusterSelectors, clusterSelectorFileObject{
				Path:            object.Path,
				ClusterSelector: o,
			})
		}
	}
	return clusterSelectors
}

// Selects returns true if the ClusterSelector selects Cluster.
// Returns an error if the LabelSelector is invalid.
func selects(cs clusterSelectorFileObject, cluster *clusterregistry.Cluster) (bool, status.Error) {
	// Convert selector preemptively, or else we won't show an error for invalid ClusterSelectors
	// if the Cluster is missing.
	selector, err := asSelector(cs)
	if err != nil {
		return false, err
	}
	if cluster == nil {
		// All ClusterSelectors are inactive if the Cluster definition is missing.
		return false, nil
	}
	return selector.Matches(labels.Set(cluster.Labels)), nil
}

// asSelector returns a known valid and nonempty label selector.
func asSelector(cs clusterSelectorFileObject) (labels.Selector, status.Error) {
	labelSelector := cs.Spec.Selector
	selector, err := metav1.LabelSelectorAsSelector(&labelSelector)
	if err != nil {
		return nil, InvalidSelectorError2(cs, err)
	}
	if selector.Empty() {
		return nil, EmptySelectorError(cs)
	}
	return selector, nil
}

// ObjectHasUnknownClusterSelectorCode is the error code for ObjectHasUnknownClusterSelector
const ObjectHasUnknownClusterSelectorCode = "1013"

var objectHasUnknownClusterSelector = status.NewErrorBuilder(ObjectHasUnknownClusterSelectorCode)

// ObjectHasUnknownClusterSelector is an error denoting an object that has an unknown annotation.
func ObjectHasUnknownClusterSelector(resource id.Resource, annotation string) status.Error {
	return objectHasUnknownClusterSelector.
		Sprintf("Resource %q MUST refer to an existing ClusterSelector, but has annotation %s=%q which maps to no declared ClusterSelector",
			resource.GetName(), v1.ClusterSelectorAnnotationKey, annotation).
		BuildWithResources(resource)
}
