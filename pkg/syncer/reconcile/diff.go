package reconcile

import (
	"context"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/syncer/differ"
	"github.com/google/nomos/pkg/syncer/labeling"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/record"
)

func handleDiff(ctx context.Context, applier Applier, diff *differ.Diff, recorder record.EventRecorder) (bool, id.ResourceError) {
	removeEmptyRulesField(diff.Declared)

	switch diff.Type() {
	case differ.NoOp:
		return false, nil
	case differ.Create:
		annotate(diff.Declared, kv(v1.ResourceManagementKey, v1.ResourceManagementEnabled))
		return applier.Create(ctx, diff.Declared)
	case differ.Update:
		annotate(diff.Declared, kv(v1.ResourceManagementKey, v1.ResourceManagementEnabled))
		return applier.Update(ctx, diff.Declared, diff.Actual)
	case differ.Delete:
		return applier.Delete(ctx, diff.Actual)
	case differ.Unmanage:
		// The intended state of an unmanaged resource is a copy of the resource, but without management enabled.
		intended := diff.Actual.DeepCopy()
		removeNomosMeta(intended)
		return applier.Update(ctx, intended, diff.Actual)
	case differ.Error:
		warnInvalidAnnotationResource(recorder, diff.Declared)
		return false, nil
	}

	panic(vet.InternalErrorf("programmatic error, unhandled syncer diff type: %v", diff.Type()))
}

func warnInvalidAnnotationResource(recorder record.EventRecorder, declared *unstructured.Unstructured) {
	err := vet.IllegalManagementAnnotationError{
		Resource: ast.ParseFileObject(declared),
		Value:    declared.GetAnnotations()[v1.ResourceManagementKey],
	}
	glog.Warning(err)
	recorder.Event(declared, corev1.EventTypeWarning, "InvalidAnnotation", err.Error())
}

func removeNomosMeta(obj metav1.Object) {
	a := obj.GetAnnotations()
	v1.RemoveNomos(a)
	obj.SetAnnotations(a)

	l := obj.GetLabels()
	labeling.RemoveQuota(l)
	obj.SetLabels(l)
}
