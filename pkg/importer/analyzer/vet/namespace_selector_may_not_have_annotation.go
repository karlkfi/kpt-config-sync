package vet

import (
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/status"
)

// NamespaceSelectorMayNotHaveAnnotationCode is the error code for NamespaceSelectorMayNotHaveAnnotation
const NamespaceSelectorMayNotHaveAnnotationCode = "1012"

func init() {
	r := role()
	r.SetName("selector")
	status.AddExamples(NamespaceSelectorMayNotHaveAnnotationCode, NamespaceSelectorMayNotHaveAnnotation(
		r,
	))
}

var namespaceSelectorMayNotHaveAnnotation = status.NewErrorBuilder(NamespaceSelectorMayNotHaveAnnotationCode)

// NamespaceSelectorMayNotHaveAnnotation reports that a namespace selector has
// an annotation that is not allowed.
func NamespaceSelectorMayNotHaveAnnotation(object core.Object) status.Error {
	// TODO(willbeason): Print information about the object so it can actually be found.
	return namespaceSelectorMayNotHaveAnnotation.Errorf("The NamespaceSelector config %q MUST NOT have ClusterSelector annotation", object.GetName())
}
