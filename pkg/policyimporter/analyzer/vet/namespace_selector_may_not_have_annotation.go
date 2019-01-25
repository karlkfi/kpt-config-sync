package vet

import "k8s.io/apimachinery/pkg/apis/meta/v1"

// NamespaceSelectorMayNotHaveAnnotationCode is the error code for NamespaceSelectorMayNotHaveAnnotation
const NamespaceSelectorMayNotHaveAnnotationCode = "1012"

func init() {
	register(NamespaceSelectorMayNotHaveAnnotationCode, nil, "")
}

// NamespaceSelectorMayNotHaveAnnotation reports that a namespace selector has
// an annotation that is not allowed.
type NamespaceSelectorMayNotHaveAnnotation struct {
	Object v1.Object
}

// Error implements error.
func (e NamespaceSelectorMayNotHaveAnnotation) Error() string {
	// TODO(willbeason): Print information about the object so it can actually be found.
	return format(e, "The NamespaceSelector Resource %q MUST NOT have ClusterSelector annotation", e.Object.GetName())
}

// Code implements Error
func (e NamespaceSelectorMayNotHaveAnnotation) Code() string {
	return NamespaceSelectorMayNotHaveAnnotationCode
}
