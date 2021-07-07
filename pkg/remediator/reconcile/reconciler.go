package reconcile

import (
	"context"

	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/diff"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/metrics"
	"github.com/google/nomos/pkg/status"
	syncerreconcile "github.com/google/nomos/pkg/syncer/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type reconcilerInterface interface {
	Remediate(ctx context.Context, id core.ID, obj client.Object) status.Error
	GetClient() client.Client
}

// reconciler ensures objects are consistent with their declared state in the
// repository.
type reconciler struct {
	scope declared.Scope
	// applier is where to write the declared configuration to.
	applier syncerreconcile.Applier
	// declared is the threadsafe in-memory representation of declared configuration.
	declared *declared.Resources
}

// newReconciler instantiates a new reconciler.
func newReconciler(
	scope declared.Scope,
	applier syncerreconcile.Applier,
	declared *declared.Resources,
) *reconciler {
	return &reconciler{
		scope:    scope,
		applier:  applier,
		declared: declared,
	}
}

// Remediate takes a client.Object representing the object to update, and then
// ensures that the version on the server matches it.
func (r *reconciler) Remediate(ctx context.Context, id core.ID, obj client.Object) status.Error {
	declU, found := r.declared.Get(id)
	// Yes, this if block is necessary because Go is pedantic about nil interfaces.
	// 1) var decl client.Object = declU results in a panic.
	// 2) Using declU as a client.Object results in a panic.
	var decl client.Object
	if found {
		decl = declU
	}

	d := diff.Diff{
		Declared: decl,
		Actual:   obj,
	}
	switch t := d.Operation(ctx, r.scope); t {
	case diff.NoOp:
		return nil
	case diff.Create:
		glog.V(3).Infof("The remediator is about to create object %v", core.GKNN(declU))
		_, err := r.applier.Create(ctx, declU)
		return err
	case diff.Update:
		actual, err := d.UnstructuredActual()
		if err != nil {
			return err
		}
		glog.V(3).Infof("The remediator is about to update object %v", core.GKNN(actual))
		_, err = r.applier.Update(ctx, declU, actual)
		return err
	case diff.Delete:
		actual, err := d.UnstructuredActual()
		if err != nil {
			return err
		}
		glog.V(3).Infof("The remediator is about to delete object %v", core.GKNN(actual))
		_, err = r.applier.Delete(ctx, actual)
		return err
	case diff.Error:
		// This is the case where the annotation in the *repository* is invalid.
		// Should never happen as the Parser would have thrown an error.
		return nonhierarchical.IllegalManagementAnnotationError(
			d.Declared,
			d.Declared.GetAnnotations()[v1.ResourceManagementKey],
		)
	case diff.Unmanage:
		actual, err := d.UnstructuredActual()
		if err != nil {
			return err
		}
		glog.V(3).Infof("The remediator is about to unmanage object %v", core.GKNN(actual))
		_, err = r.applier.RemoveNomosMeta(ctx, actual)
		return err
	default:
		// e.g. differ.DeleteNsConfig, which shouldn't be possible to get to any way.
		metrics.RecordInternalError(ctx, "remediator")
		return status.InternalErrorf("diff type not supported: %v", t)
	}
}

// GetClient returns the reconciler's underlying client.Client.
func (r *reconciler) GetClient() client.Client {
	return r.applier.GetClient()
}
