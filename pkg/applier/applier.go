package applier

import (
	"context"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/syncer/differ"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/status"

	syncerreconcile "github.com/google/nomos/pkg/syncer/reconcile"
)

// Applier declares the Applier component in the Multi Repo Reconciler Process.
type Applier struct {
	// applier provides the basic resource creation, updating and deletion functions.
	applier syncerreconcile.Applier
	// cachedResources stores the previously parsed resources in memory. The applier uses this
	// cachedResources to compare the previous resource (actual) with the new parsed ones (declared)
	// and get the diff.
	cachedResources map[core.ID]ast.FileObject
}

// New initializes an Applier component.
func New(applier syncerreconcile.Applier) *Applier {
	return &Applier{applier: applier, cachedResources: make(map[core.ID]ast.FileObject)}
}

// Apply iterates all the resources in a cluster and syncs the resource with its git repo state.
func (a *Applier) Apply(ctx context.Context, declaredResources []ast.FileObject) error {
	// TODO: For the very first run, the cachedResources would be empty and couldn't reflect
	// the resource state. Thus, it should be built from the API server before comparing with
	// the declared resources.
	if len(a.cachedResources) == 0 {
		if err := a.sync(); err != nil {
			return errors.Wrap(err, "failed to sync the cachedResources from API server")
		}
	}
	newCached := make(map[core.ID]ast.FileObject)
	if len(declaredResources) == 0 {
		glog.V(4).Infof("no declared resources to apply.")
	}
	// Take CRUD actions based on the diff between actual resource (cached, reflecting what the
	// api server stores) and the declared resource (reflecting the real git repo).
	var errs status.MultiError
	for _, declared := range declaredResources {
		d, err := a.diff(declared)
		if err != nil {
			errs = status.Append(errs, err)
			continue
		}
		coreID := core.IDOf(declared)
		// Note: coreID is not types.UID (a cluster-scoped unique string), but a GVKNN.
		newCached[coreID] = declared
		switch d.Type() {
		case differ.NoOp:
			continue
		case differ.Error:
			err = nonhierarchical.IllegalManagementAnnotationError(declared,
				declared.GetAnnotations()[v1.ResourceManagementKey])
			errs = status.Append(errs, err)
		case differ.Create:
			if _, e := a.applier.Create(ctx, d.Declared); e != nil {
				err = status.ResourceWrap(e, "unable to create resource %s", declared)
				errs = status.Append(errs, err)
			} else {
				glog.V(4).Infof("created resource %s", coreID.String())
			}
		case differ.Update:
			if _, e := a.applier.Update(ctx, d.Declared, d.Actual); e != nil {
				err = status.ResourceWrap(e, "unable to update resource %s", declared)
				errs = status.Append(errs, err)
			} else {
				glog.V(4).Infof("updated resource %s", coreID.String())
			}
		case differ.Delete:
			// Note: Since the for loop iterates on the declaredResources, this case shall not
			// be triggered unless the passed in declaredResources contains nil elements.
			if _, e := a.applier.Delete(ctx, d.Actual); e != nil {
				err = status.ResourceWrap(e, "unable to delete %s", declared)
				errs = status.Append(errs, err)
			} else {
				glog.V(4).Infof("deleted resource %s", coreID.String())
			}
		case differ.Unmanage:
			if _, e := a.applier.RemoveNomosMeta(ctx, d.Actual); e != nil {
				err = status.ResourceWrap(
					e, "unable to remove the nomos meta from %s", declared)
				errs = status.Append(errs, err)
			} else {
				glog.V(4).Infof("unmanaged the resource %s", coreID.String())
			}
		default:
			err = status.InternalErrorf("diff type not supported: %v", d.Type())
			errs = status.Append(errs, err)
		}
	}
	// Prune the actual resource if they no longer exist in the new declared resource list.
	for coreID, actual := range a.cachedResources {
		if _, ok := newCached[coreID]; !ok {
			cachedActual, err := syncerreconcile.AsUnstructured(actual.Object)
			if err != nil {
				err := syntax.ObjectParseError(actual, err)
				errs = status.Append(errs, err)
				continue
			}
			if _, e := a.applier.Delete(ctx, cachedActual); e != nil {
				err := errors.Wrapf(e, "unable to delete resource %s", coreID.String())
				errs = status.Append(errs, err)
			} else {
				glog.V(4).Infof("deleted resource %s", coreID.String())
			}
		}
	}
	// TODO: make a table/list of all of the failure modes and have an action for each one.
	// e.g. the declared state is not really applied to the resource, we shall not update the cache.
	a.cachedResources = newCached
	glog.Infof("applier has synced all resources.")
	return errs
}

// diff builds a Diff struct from the declared resource and the cached resources (if exists).
func (a *Applier) diff(declared ast.FileObject) (*differ.Diff, status.Error) {
	decl, err := syncerreconcile.AsUnstructured(declared.Object)
	if err != nil {
		return &differ.Diff{}, syntax.ObjectParseError(declared, err)
	}
	uid := core.IDOf(decl)
	var actual *unstructured.Unstructured
	if cached, ok := a.cachedResources[uid]; ok {
		actual, err = syncerreconcile.AsUnstructured(cached.Object)
		if err != nil {
			return &differ.Diff{}, syntax.ObjectParseError(cached, err)
		}
	}
	return &differ.Diff{
		Name:     uid.String(),
		Actual:   actual,
		Declared: decl,
	}, nil
}

// sync pulls the stored resources from the API server and builds up the cachedResources.
// TODO
func (a *Applier) sync() error { return nil }
