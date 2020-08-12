package applier

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/diff"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/importer/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/status"
	syncerreconcile "github.com/google/nomos/pkg/syncer/reconcile"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Applier declares the Applier component in the Multi Repo Reconciler Process.
type Applier struct {
	// applier provides the basic resource creation, updating and deletion functions.
	applier syncerreconcile.Applier
	// cachedObjects stores the previously parsed git resources in memory. The applier uses this
	// cachedResources to compare with newly parsed resources from git to determine which
	// previously declared resources should be deleted.
	cachedObjects map[core.ID]core.Object
	// reader reads and lists the resources from API server.
	reader client.Reader
	// listOptions defines the resource filtering condition for different appliers initialized
	// by root reconcile process or namespace reconcile process.
	listOptions []client.ListOption
}

// Interface is a fake-able subset of the interface Applier implements.
//
// Placed here to make discovering the production implementation (above) easier.
type Interface interface {
	Apply(ctx context.Context, desiredResource []core.Object) error
}

// NewNamespaceApplier initializes an applier that fetches a certain namespace's resources from
// the API server.
func NewNamespaceApplier(reader client.Reader,
	applier syncerreconcile.Applier, namespace string) *Applier {
	// TODO(b/161256730): Constrains the resources due to the new labeling strategy.
	opts := []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabels{v1.ManagedByKey: v1.ManagedByValue}}
	a := &Applier{
		listOptions: opts, reader: reader, applier: applier,
		cachedObjects: make(map[core.ID]core.Object)}
	glog.V(4).Infof("Applier %v is initialized", namespace)
	return a
}

// NewRootApplier initializes an applier that can fetch all resources from the API server.
func NewRootApplier(reader client.Reader, applier syncerreconcile.Applier) *Applier {
	// TODO(b/161256730): Constrains the resources due to the new labeling strategy.
	opts := []client.ListOption{
		client.MatchingLabels{v1.ManagedByKey: v1.ManagedByValue}}
	a := &Applier{
		listOptions: opts, reader: reader, applier: applier,
		cachedObjects: make(map[core.ID]core.Object)}
	glog.V(4).Infof("Root applier is initialized and synced with the API server")
	return a
}

// sync actuates a list of Diff to make sure the actual resources in the API service are
// in sync with the declared resources.
func (a *Applier) sync(ctx context.Context, diffs []diff.Diff) error {
	var errs status.MultiError
	for _, d := range diffs {
		// Take CRUD actions based on the diff between actual resource (what's stored in
		// the api server) and the declared resource (the cached git resource).
		var decl core.Object
		if d.Declared != nil {
			decl = d.Declared.DeepCopyObject().(core.Object)
		} else {
			decl = d.Actual.DeepCopyObject().(core.Object)
		}
		coreID := core.IDOf(decl)

		switch d.Type() {
		case diff.NoOp:
			continue
		case diff.Error:
			err := nonhierarchical.IllegalManagementAnnotationError(decl,
				decl.GetAnnotations()[v1.ResourceManagementKey])
			errs = status.Append(errs, err)
		case diff.Create:
			declared, err := d.UnstructuredDeclared()
			if err != nil {
				errs = status.Append(errs, syntax.ObjectParseError(d.Declared, err))
				continue
			}
			if _, e := a.applier.Create(ctx, declared); e != nil {
				err := status.ResourceWrap(e, "unable to create resource %s", decl)
				errs = status.Append(errs, err)
			} else {
				glog.V(4).Infof("created resource %v", coreID)
			}
		case diff.Update:
			declared, err := d.UnstructuredDeclared()
			if err != nil {
				errs = status.Append(errs, syntax.ObjectParseError(d.Declared, err))
				continue
			}
			actual, err := d.UnstructuredActual()
			if err != nil {
				errs = status.Append(errs, syntax.ObjectParseError(d.Actual, err))
				continue
			}
			if _, e := a.applier.Update(ctx, declared, actual); e != nil {
				err := status.ResourceWrap(e, "unable to update resource %s", decl)
				errs = status.Append(errs, err)
			} else {
				glog.V(4).Infof("updated resource %v", coreID)
			}
		case diff.Delete:
			actual, err := d.UnstructuredActual()
			if err != nil {
				errs = status.Append(errs, syntax.ObjectParseError(d.Actual, err))
				continue
			}
			if _, e := a.applier.Delete(ctx, actual); e != nil {
				err := status.ResourceWrap(e, "unable to delete %s", decl)
				errs = status.Append(errs, err)
			} else {
				glog.V(4).Infof("deleted resource %v", coreID)
			}
		case diff.Unmanage:
			actual, err := d.UnstructuredActual()
			if err != nil {
				errs = status.Append(errs, syntax.ObjectParseError(d.Actual, err))
				continue
			}
			if _, e := a.applier.RemoveNomosMeta(ctx, actual); e != nil {
				err := status.ResourceWrap(
					e, "unable to remove the nomos meta from %v", decl)
				errs = status.Append(errs, err)
			} else {
				glog.V(4).Infof("unmanaged the resource %v", coreID)
			}
		default:
			err := status.InternalErrorf("diff type not supported: %v", d.Type())
			errs = status.Append(errs, err)
		}
	}
	glog.V(4).Infof("all resources are up to date.")
	return errs
}

// Apply updates the resource API server with the latest parsed git resource. This is called
// when a new change in the git resource is detected.
func (a *Applier) Apply(ctx context.Context, desiredResource []core.Object) error {
	// create the new cache showing the new declared resource.
	newCache := make(map[core.ID]core.Object)
	for _, desired := range desiredResource {
		newCache[core.IDOf(desired)] = desired
	}

	// pull the actual resource from the API server.
	actualObjects, err := a.getActualObject(ctx)
	if err != nil {
		return err
	}
	diffs := diff.ThreeWay(newCache, a.cachedObjects, actualObjects)
	// Sync the API resource state to the git resource.
	if err := a.sync(ctx, diffs); err != nil {
		return err
	}
	// Update the cache.
	a.cachedObjects = newCache
	return nil
}

// Refresh syncs and updates the API server with the (cached) git resource states.
func (a *Applier) Refresh(ctx context.Context) error {
	actualObjects, err := a.getActualObject(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to pull the resource from the API server")

	}
	// Two way merge. Compare between the cached declared and the actual states to decide the create and update.
	diffs := diff.TwoWay(a.cachedObjects, actualObjects)
	if err := a.sync(ctx, diffs); err != nil {
		return fmt.Errorf("applier failure: %v", err)
	}
	return nil
}

// getActualObject pulls the stored resource from the API server.
func (a *Applier) getActualObject(ctx context.Context) (map[core.ID]core.Object, error) {
	resources := &unstructured.UnstructuredList{}
	if err := a.reader.List(ctx, resources, a.listOptions...); err != nil {
		return nil, status.APIServerError(err, "failed to list resources")
	}
	actual := make(map[core.ID]core.Object)
	var errs status.MultiError
	for _, res := range resources.Items {
		obj := res.DeepCopyObject().(core.Object)
		if coreID, err := core.IDOfRuntime(obj); err != nil {
			errs = status.Append(errs, err)
		} else {
			actual[coreID] = obj
		}
	}
	return actual, errs
}
