package veterrors

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ObjectHasUnknownClusterSelector is an error denoting an object that has an unknown annotation.
type ObjectHasUnknownClusterSelector struct {
	Object     v1.Object
	Annotation string
}

// Error implements error.
func (e ObjectHasUnknownClusterSelector) Error() string {
	return format(e, "Resource %q MUST refer to an existing ClusterSelector, but has annotation %s=%q which maps to no declared ClusterSelector", e.Object.GetName(), v1alpha1.ClusterSelectorAnnotationKey, e.Annotation)
}

// Code implements Error
func (e ObjectHasUnknownClusterSelector) Code() string { return ObjectHasUnknownClusterSelectorCode }
