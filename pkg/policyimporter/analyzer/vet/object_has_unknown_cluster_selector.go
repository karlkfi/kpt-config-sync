package vet

import (
	v12 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ObjectHasUnknownClusterSelectorCode is the error code for ObjectHasUnknownClusterSelector
const ObjectHasUnknownClusterSelectorCode = "1013"

func init() {
	register(ObjectHasUnknownClusterSelectorCode, nil, "")
}

// ObjectHasUnknownClusterSelector is an error denoting an object that has an unknown annotation.
type ObjectHasUnknownClusterSelector struct {
	Object     v1.Object
	Annotation string
}

// Error implements error.
func (e ObjectHasUnknownClusterSelector) Error() string {
	return format(e, "Resource %q MUST refer to an existing ClusterSelector, but has annotation %s=%q which maps to no declared ClusterSelector", e.Object.GetName(), v12.ClusterSelectorAnnotationKey, e.Annotation)
}

// Code implements Error
func (e ObjectHasUnknownClusterSelector) Code() string { return ObjectHasUnknownClusterSelectorCode }
