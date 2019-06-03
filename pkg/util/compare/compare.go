package compare

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// MetaEqual returns true if left objects's labels and annotations are equal to labels and
// annotations right object.
func MetaEqual(left metav1.Object, right metav1.Object) bool {
	return reflect.DeepEqual(left.GetLabels(), right.GetLabels()) && reflect.DeepEqual(left.GetAnnotations(), right.GetAnnotations())
}

// ObjectMetaEqual returns true if the Meta field of left and right objects are equal.
func ObjectMetaEqual(left runtime.Object, right runtime.Object) bool {
	return MetaEqual(left.(metav1.Object), right.(metav1.Object))
}
