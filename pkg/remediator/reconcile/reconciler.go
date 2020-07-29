package reconcile

import (
	"context"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/parse/declaredresources"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/differ"
	syncerreconcile "github.com/google/nomos/pkg/syncer/reconcile"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// reconciler ensures objects are consistent with their declared state in the
// repository.
type reconciler struct {
	// reader is the source of "actual" configuration on a Kubernetes cluster.
	reader client.Reader
	// applier is where to write the declared configuration to.
	applier syncerreconcile.Applier
	// declared is the threadsafe in-memory representation of declared configuration.
	declared *declaredresources.DeclaredResources
}

// newReconciler instantiates a new reconciler.
func newReconciler(
	reader client.Reader,
	applier syncerreconcile.Applier,
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
func (r *reconciler) Remediate(ctx context.Context, id core.ID, obj core.Object) error {
	diff, err := r.diff(id, obj)
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

func (r *reconciler) diff(id core.ID, obj core.Object) (*differ.Diff, error) {
	var actual *unstructured.Unstructured
	var err error
	if obj == nil {
		actual = nil
	} else {
		actual, err = syncerreconcile.AsUnstructuredSanitized(obj)
		if err != nil {
			return nil, err
		}
	}

	declared, _ := r.declared.GetDecl(id)
	return &differ.Diff{
		Name:     id.Name,
		Declared: declared,
		Actual:   actual,
	}, nil
}
