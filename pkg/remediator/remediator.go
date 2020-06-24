package remediator

import (
	"context"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/parse/declaredresources"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/differ"
	"github.com/google/nomos/pkg/syncer/reconcile"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Remediator ensures objects are consistent with their declared state in the
// repository.
type Remediator struct {
	// reader is the source of "actual" configuration on a Kubernetes cluster.
	reader client.Reader
	// applier is where to write the declared configuration to.
	applier reconcile.Applier
	// declared is the threadsafe in-memory representation of declared configuration.
	declared *declaredresources.DeclaredResources
}

// New instantiates a new Remediator.
func New(
	reader client.Reader,
	applier reconcile.Applier,
	declared *declaredresources.DeclaredResources,
) *Remediator {
	return &Remediator{
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
func (r *Remediator) Remediate(ctx context.Context, obj runtime.Object) error {
	diff, err := r.diff(ctx, obj)
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
		// TODO(b/157587458): Don't delete special Namespaces.
		_, err := r.applier.Delete(ctx, diff.Actual)
		return err
	case differ.Error:
		// This is the case where the annotation in the *repository* is invalid.
		// Should never happen as the Parser would have thrown an error.
		return nonhierarchical.IllegalManagementAnnotationError(
			ast.ParseFileObject(diff.Declared),
			diff.Declared.GetAnnotations()[v1.ResourceManagementKey],
		)
	case differ.Unmanage:
		_, err := r.applier.RemoveNomosMeta(ctx, diff.Actual)
		return err
	default:
		// e.g. differ.DeleteNsConfig, which shouldn't be possible to get to any way.
		return status.InternalErrorf("diff type not supported: %v", diff.Type())
	}
}

func (r *Remediator) diff(ctx context.Context, obj runtime.Object) (*differ.Diff, error) {
	id, err := core.IDOfRuntime(obj)
	if err != nil {
		return nil, errors.Wrap(err, "getting ID of object to reconcile")
	}

	declared, isDeclared := r.declared.GetDecl(id)
	actual := &unstructured.Unstructured{}
	if !isDeclared {
		declared = nil
		actual.SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())
	} else {
		actual.SetGroupVersionKind(declared.GroupVersionKind())
	}

	err = r.reader.Get(ctx, id.ObjectKey, actual)
	switch {
	case err == nil:
		// We isDeclared the object on the cluster.
	case apierrors.IsNotFound(err):
		actual = nil
	default:
		return nil, err
	}

	return &differ.Diff{
		Declared: declared,
		Actual:   actual,
	}, nil
}
