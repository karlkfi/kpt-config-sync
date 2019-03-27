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

	managementState := diff.ManagementState()
	if managementState == differ.Invalid {
		// The resource's management state is not valid, so show an error and do nothing.
		warnInvalidAnnotationResource(recorder, diff.Actual, "declared")
		return false, nil
	}

	if diff.Type == differ.Create {
		// It isn't currently possible for a resource to not exist on the cluster and be explicitly
		// unmanaged. The logic is currently this way because Diff.ManagementState treats
		// "the resource does not exist on the cluster" as "the resource is not managed".
		// This replicates preexisting behavior, but will be changed with b/129358726.
		return applier.Create(ctx, diff.Declared)
	}

	if managementState == differ.Unmanaged {
		// The resource is explicitly marked unmanaged, so do nothing.
		return false, nil
	}

	switch diff.Type {
	case differ.Update:
		// The resource is either "managed", or has no management annotation and
		// is in the repo, so update it.
		return applier.Update(ctx, diff.Declared, diff.Actual)

	case differ.Delete:
		if managementState == differ.Unset {
			// Do not delete resource if managed annotation is unset.
			return false, nil
		}
		return applier.Delete(ctx, diff.Actual)
	}
	panic(vet.InternalErrorf("programmatic error, unhandled syncer diff type combination: %v and %v", diff.Type, managementState))
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
