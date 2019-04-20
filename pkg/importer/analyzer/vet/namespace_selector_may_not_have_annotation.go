package vet

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NamespaceSelectorMayNotHaveAnnotationCode is the error code for NamespaceSelectorMayNotHaveAnnotation
const NamespaceSelectorMayNotHaveAnnotationCode = "1012"

func init() {
	r := role()
	r.MetaObject().SetName("selector")
	status.Register(NamespaceSelectorMayNotHaveAnnotationCode, NamespaceSelectorMayNotHaveAnnotation{
		Object: r.MetaObject(),
	})
}

// NamespaceSelectorMayNotHaveAnnotation reports that a namespace selector has
// an annotation that is not allowed.
type NamespaceSelectorMayNotHaveAnnotation struct {
	Object metav1.Object
}

// Error implements error.
func (e NamespaceSelectorMayNotHaveAnnotation) Error() string {
	// TODO(willbeason): Print information about the object so it can actually be found.
	return status.Format(e, "The NamespaceSelector config %q MUST NOT have ClusterSelector annotation", e.Object.GetName())
}

// Code implements Error
func (e NamespaceSelectorMayNotHaveAnnotation) Code() string {
	return NamespaceSelectorMayNotHaveAnnotationCode
}

// ToCME implements ToCMEr.
func (e NamespaceSelectorMayNotHaveAnnotation) ToCME() v1.ConfigManagementError {
	return status.FromError(e)
}
