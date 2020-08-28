package reconcile

import (
	"context"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/diff"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/status"
	syncerreconcile "github.com/google/nomos/pkg/syncer/reconcile"
)

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

// Remediate takes an runtime.Object representing the object to update, and then
// ensures that the version on the server matches it.
func (r *reconciler) Remediate(ctx context.Context, id core.ID, obj core.Object) error {
	declU, _ := r.declared.Get(id)
	var decl core.Object
	var err error
	if declU == nil {
		decl = nil
	} else {
		decl, err = core.ObjectOf(declU)
		if err != nil {
			return err
		}
	}

	d := diff.Diff{
		Name:     id.Name,
		Declared: decl,
		Actual:   obj,
	}
	switch t := d.Type(r.scope); t {
	case diff.NoOp:
		return nil
	case diff.Create:
		_, err := r.applier.Create(ctx, declU)
		return err
	case diff.Update:
		actual, err := d.UnstructuredActual()
		if err != nil {
			return err
		}
		_, err = r.applier.Update(ctx, declU, actual)
		return err
	case diff.Delete:
		actual, err := d.UnstructuredActual()
		if err != nil {
			return err
		}
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
		_, err = r.applier.RemoveNomosMeta(ctx, actual)
		return err
	default:
		// e.g. differ.DeleteNsConfig, which shouldn't be possible to get to any way.
		return status.InternalErrorf("diff type not supported: %v", t)
	}
}
