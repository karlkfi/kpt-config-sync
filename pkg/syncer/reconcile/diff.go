package reconcile

import (
	"context"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/syncer/differ"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/record"
)

func handleDiff(ctx context.Context, applier Applier, diff *differ.Diff, recorder record.EventRecorder) (bool, id.ResourceError) {
	removeEmptyRulesField(diff.Declared)

	switch diff.Type() {
	case differ.NoOp:
		return false, nil
	case differ.Create:
		return applier.Create(ctx, diff.Declared)
	case differ.Update:
		return applier.Update(ctx, diff.Declared, diff.Actual)
	case differ.Delete:
		return applier.Delete(ctx, diff.Actual)
	case differ.Error:
		warnInvalidAnnotationResource(recorder, diff.Actual, "declared")
		return false, nil
	}

	panic(vet.InternalErrorf("programmatic error, unhandled syncer diff type: %v", diff.Type()))
}

func warnInvalidAnnotationResource(recorder record.EventRecorder, u *unstructured.Unstructured, msg string) {
	gvk := u.GroupVersionKind()
	value := u.GetAnnotations()[v1.ResourceManagementKey]
	glog.Warningf("%q with name %q is %s in the source of truth but has invalid management annotation %s=%s",
		gvk, u.GetName(), msg, v1.ResourceManagementKey, value)
	recorder.Eventf(
		u, corev1.EventTypeWarning, "InvalidAnnotation",
		"%q is %s in the source of truth but has invalid management annotation %s=%s", gvk, v1.ResourceManagementKey, value)
}
