package vet

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/policyimporter/id"
	"github.com/google/nomos/pkg/status"
)

// ObjectHasUnknownClusterSelectorCode is the error code for ObjectHasUnknownClusterSelector
const ObjectHasUnknownClusterSelectorCode = "1013"

func init() {
	status.Register(ObjectHasUnknownClusterSelectorCode, ObjectHasUnknownClusterSelector{
		Resource:   role(),
		Annotation: "non-existent-cluster",
	})
}

// ObjectHasUnknownClusterSelector is an error denoting an object that has an unknown annotation.
type ObjectHasUnknownClusterSelector struct {
	id.Resource
	Annotation string
}

// Error implements error.
func (e ObjectHasUnknownClusterSelector) Error() string {
	return status.Format(e, "Resource %q MUST refer to an existing ClusterSelector, but has annotation %s=%q which maps to no declared ClusterSelector", e.Name(), v1.ClusterSelectorAnnotationKey, e.Annotation)
}

// Code implements Error
func (e ObjectHasUnknownClusterSelector) Code() string { return ObjectHasUnknownClusterSelectorCode }
