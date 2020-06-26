package remediator

import (
	"context"

	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/parse/declaredresources"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/differ"
	"github.com/google/nomos/pkg/syncer/reconcile"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// reconciler ensures objects are consistent with their declared state in the
// repository.
type reconciler struct {
	// reader is the source of "actual" configuration on a Kubernetes cluster.
	reader client.Reader
	// applier is where to write the declared configuration to.
	applier reconcile.Applier
	// declared is the threadsafe in-memory representation of declared configuration.
	declared *declaredresources.DeclaredResources
}

// newReconciler instantiates a new reconciler.
func newReconciler(
	reader client.Reader,
	applier reconcile.Applier,
	declared *declaredresources.DeclaredResources,
) *reconciler {
	return &reconciler{
		reader:   reader,
		applier:  applier,
		declared: declared,
	}
}

// Remediate takes an runtime.Object representing the object to update, and then
// ensures that the version on the server matches it.
//
// core.ID doesn't record the Version, and we need the Version in order to
// retrieve the object's current state from the cluster.
func (r *reconciler) Remediate(ctx context.Context, gvknn GVKNN) error {
	diff, err := r.diff(ctx, gvknn)
	if err != nil {
		return err
	}

	switch diff.Type() {
	case differ.NoOp:
		return nil
	case differ.Create:
		_, err := r.applier.Create(ctx, diff.Declared)
		return err
	case differ.Update:
		_, err := r.applier.Update(ctx, diff.Declared, diff.Actual)
		return err
	case differ.Delete:
		_, err := r.applier.Delete(ctx, diff.Actual)
		return err
	case differ.Error:
		// This is the case where the annotation in the *repository* is invalid.
		// Should never happen as the Parser would have thrown an error.
		return nonhierarchical.IllegalManagementAnnotationError(
			ast.ParseFileObject(diff.Declared),
			diff.Declared.GetAnnotations()[v1.ResourceManagementKey],
		)
	case differ.Unmanage, differ.UnmanageSystemNamespace:
		_, err := r.applier.RemoveNomosMeta(ctx, diff.Actual)
		return err
	default:
		// e.g. differ.DeleteNsConfig, which shouldn't be possible to get to any way.
		return status.InternalErrorf("diff type not supported: %v", diff.Type())
	}
}

func (r *reconciler) diff(ctx context.Context, gvknn GVKNN) (*differ.Diff, error) {
	id := gvknn.ID

	declared, isDeclared := r.declared.GetDecl(id)
	actual := &unstructured.Unstructured{}
	switch {
	case !isDeclared:
		// Not declared in the SOT, so use the requested version.
		declared = nil
		actual.SetGroupVersionKind(gvknn.GroupVersionKind())
	case gvknn.GroupVersionKind() != declared.GroupVersionKind():
		// TODO(b/159723324): Drop this case since WatchManager won't start watches
		//  for multiple versions of the same GK.
		glog.V(4).Infof("ignored version %q inconsistent with %v", gvknn.Version, id)
		// Trying to remediate a different version than the declared; ignore.
		return &differ.Diff{}, nil
	default:
		// The requested/declared versions match.
		actual.SetGroupVersionKind(declared.GroupVersionKind())
	}
	err := r.reader.Get(ctx, id.ObjectKey, actual)

	switch {
	case err == nil:
		// We found the object on the cluster.
	case apierrors.IsNotFound(err):
		actual = nil
	default:
		return nil, err
	}

	return &differ.Diff{
		Name:     id.Name,
		Declared: declared,
		Actual:   actual,
	}, nil
}
