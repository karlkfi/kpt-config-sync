// Package util contains shared functionality for constraints and constraint templates.
package util

import (
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/metadata"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// AnnotateErrors sets the error status annotation to the given error messages.
func AnnotateErrors(obj *unstructured.Unstructured, msgs ...string) {
	core.SetAnnotation(obj, metadata.ResourceStatusErrorsKey, jsonify(msgs))
}

// AnnotateReconciling sets the reconciling status annotation to the given reasons.
func AnnotateReconciling(obj *unstructured.Unstructured, msgs ...string) {
	core.SetAnnotation(obj, metadata.ResourceStatusReconcilingKey, jsonify(msgs))
}

// ResetAnnotations removes all status annotations.
func ResetAnnotations(obj *unstructured.Unstructured) {
	core.RemoveAnnotations(obj, metadata.ResourceStatusReconcilingKey)
	core.RemoveAnnotations(obj, metadata.ResourceStatusErrorsKey)
}

// AnnotationsChanged returns true if the status annotations between the two resources.
func AnnotationsChanged(newObj, oldObj *unstructured.Unstructured) bool {
	newAnns := newObj.GetAnnotations()
	oldAnns := oldObj.GetAnnotations()
	return newAnns[metadata.ResourceStatusReconcilingKey] != oldAnns[metadata.ResourceStatusReconcilingKey] ||
		newAnns[metadata.ResourceStatusErrorsKey] != oldAnns[metadata.ResourceStatusErrorsKey]
}
