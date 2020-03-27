// Package util contains shared functionality for constraints and constraint templates.
package util

import (
	nomosv1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// AnnotateErrors sets the error status annotation to the given error messages.
func AnnotateErrors(obj *unstructured.Unstructured, msgs ...string) {
	core.SetAnnotation(obj, nomosv1.ResourceStatusErrorsKey, jsonify(msgs))
}

// AnnotateReconciling sets the reconciling status annotation to the given reasons.
func AnnotateReconciling(obj *unstructured.Unstructured, msgs ...string) {
	core.SetAnnotation(obj, nomosv1.ResourceStatusReconcilingKey, jsonify(msgs))
}

// ResetAnnotations removes all status annotations.
func ResetAnnotations(obj *unstructured.Unstructured) {
	core.RemoveAnnotations(obj, nomosv1.ResourceStatusReconcilingKey)
	core.RemoveAnnotations(obj, nomosv1.ResourceStatusErrorsKey)
}

// AnnotationsChanged returns true if the status annotations between the two resources.
func AnnotationsChanged(newObj, oldObj *unstructured.Unstructured) bool {
	newAnns := newObj.GetAnnotations()
	oldAnns := oldObj.GetAnnotations()
	return newAnns[nomosv1.ResourceStatusReconcilingKey] != oldAnns[nomosv1.ResourceStatusReconcilingKey] ||
		newAnns[nomosv1.ResourceStatusErrorsKey] != oldAnns[nomosv1.ResourceStatusErrorsKey]
}
