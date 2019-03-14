package vet

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ObjectHasUnknownClusterSelectorCode is the error code for ObjectHasUnknownClusterSelector
const ObjectHasUnknownClusterSelectorCode = "1013"

func init() {
	register(ObjectHasUnknownClusterSelectorCode)
}

// ObjectHasUnknownClusterSelector is an error denoting an object that has an unknown annotation.
type ObjectHasUnknownClusterSelector struct {
	Object     metav1.Object
	Annotation string
}

// Error implements error.
func (e ObjectHasUnknownClusterSelector) Error() string {
	return status.Format(e, "Resource %q MUST refer to an existing ClusterSelector, but has annotation %s=%q which maps to no declared ClusterSelector", e.Object.GetName(), v1.ClusterSelectorAnnotationKey, e.Annotation)
}

// Code implements Error
func (e ObjectHasUnknownClusterSelector) Code() string { return ObjectHasUnknownClusterSelectorCode }
