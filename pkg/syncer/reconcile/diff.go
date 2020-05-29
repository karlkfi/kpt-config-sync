package reconcile

import (
	"context"

	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/differ"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/record"
)

// HandleDiff updates objects on the cluster based on the difference between actual and declared resources.
func HandleDiff(ctx context.Context, applier Applier, diff *differ.Diff, recorder record.EventRecorder) (bool, status.Error) {
	switch diff.Type() {
	case differ.NoOp:
		return false, nil
	case differ.Create:
		enableManagement(diff.Declared)
		return applier.Create(ctx, diff.Declared)
	case differ.Update:
		enableManagement(diff.Declared)
		return applier.Update(ctx, diff.Declared, diff.Actual)
	case differ.Delete:
		return applier.Delete(ctx, diff.Actual)
	case differ.Unmanage:
		// The intended state of an unmanaged resource is a copy of the resource, but without management enabled.
		// See b/157751323 for context on why we are doing a specific Remove() here instead of a generic Update().
		return applier.RemoveNomosMeta(ctx, diff.Actual)
	case differ.Error:
		warnInvalidAnnotationResource(recorder, diff.Declared)
		return false, nil
	}

	panic(status.InternalErrorf("programmatic error, unhandled syncer diff type: %v", diff.Type()))
}

func warnInvalidAnnotationResource(recorder record.EventRecorder, declared *unstructured.Unstructured) {
	err := nonhierarchical.IllegalManagementAnnotationError(
		ast.ParseFileObject(declared),
		declared.GetAnnotations()[v1.ResourceManagementKey],
	)
	glog.Warning(err)
	recorder.Event(declared, corev1.EventTypeWarning, v1.EventReasonInvalidAnnotation, err.Error())
}
