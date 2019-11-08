package vet

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// ObjectHasUnknownClusterSelectorCode is the error code for ObjectHasUnknownClusterSelector
const ObjectHasUnknownClusterSelectorCode = "1013"

func init() {
	status.AddExamples(ObjectHasUnknownClusterSelectorCode, ObjectHasUnknownClusterSelector(
		role(),
		"non-existent-cluster",
	))
}

var objectHasUnknownClusterSelector = status.NewErrorBuilder(ObjectHasUnknownClusterSelectorCode)

// ObjectHasUnknownClusterSelector is an error denoting an object that has an unknown annotation.
func ObjectHasUnknownClusterSelector(resource id.Resource, annotation string) status.Error {
	return objectHasUnknownClusterSelector.WithResources(resource).Errorf(
		"Resource %q MUST refer to an existing ClusterSelector, but has annotation %s=%q which maps to no declared ClusterSelector",
		resource.GetName(), v1.ClusterSelectorAnnotationKey, annotation)
}
