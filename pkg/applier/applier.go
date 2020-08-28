package applier

import (
	"context"
	"sync"

	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/diff"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/importer/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/status"
	syncerreconcile "github.com/google/nomos/pkg/syncer/reconcile"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
	// client reads and lists the resources from API server and updates RepoSync status.
	client client.Client
	// listOptions defines the resource filtering condition for different appliers initialized
	// by root reconcile process or namespace reconcile process.
	listOptions []client.ListOption
	// scope is the scope of the applier (eg root or a namespace).
	scope declared.Scope
	// mux is an Applier-level mutext to prevent concurrent Apply() and Refresh()
	mux sync.Mutex
}

// Interface is a fake-able subset of the interface Applier implements.
//
// Placed here to make discovering the production implementation (above) easier.
type Interface interface {
	Apply(ctx context.Context, desiredResource []core.Object) status.MultiError
}

// NewNamespaceApplier initializes an applier that fetches a certain namespace's resources from
// the API server.
func NewNamespaceApplier(c client.Client, applier syncerreconcile.Applier, namespace declared.Scope) *Applier {
	// TODO(b/161256730): Constrains the resources due to the new labeling strategy.
	opts := []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabels{v1.ManagedByKey: v1.ManagedByValue}}
	a := &Applier{
		applier:       applier,
		cachedObjects: make(map[core.ID]core.Object),
		client:        c,
		listOptions:   opts,
		scope:         namespace,
	}
	glog.V(4).Infof("Applier %v is initialized", namespace)
	return a
}

// NewRootApplier initializes an applier that can fetch all resources from the API server.
func NewRootApplier(c client.Client, applier syncerreconcile.Applier) *Applier {
	// TODO(b/161256730): Constrains the resources due to the new labeling strategy.
	opts := []client.ListOption{
		client.MatchingLabels{v1.ManagedByKey: v1.ManagedByValue}}
	a := &Applier{
		applier:       applier,
		cachedObjects: make(map[core.ID]core.Object),
		client:        c,
		listOptions:   opts,
		scope:         declared.RootReconciler,
	}
	glog.V(4).Infof("Root applier is initialized and synced with the API server")
	return a
}

// sync actuates a list of Diff to make sure the actual resources in the API service are
// in sync with the declared resources.
func (a *Applier) sync(ctx context.Context, diffs []diff.Diff) status.MultiError {
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

		switch t := d.Type(a.scope); t {
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
		case diff.ManagementConflict:
			err := ManagementConflictError(d.Declared)
			errs = status.Append(errs, err)
		default:
			err := status.InternalErrorf("diff type not supported: %v", t)
			errs = status.Append(errs, err)
		}
	}
	glog.V(4).Infof("all resources are up to date.")
	return errs
}

// Apply updates the resource API server with the latest parsed git resource. This is called
// when a new change in the git resource is detected.
func (a *Applier) Apply(ctx context.Context, desiredResource []core.Object) status.MultiError {
	// create the new cache showing the new declared resource.
	newCache := make(map[core.ID]core.Object)
	gvks := make(map[schema.GroupVersionKind]bool)
	for _, desired := range desiredResource {
		newCache[core.IDOf(desired)] = desired
		gvks[desired.GroupVersionKind()] = true
	}

	a.mux.Lock()
	defer a.mux.Unlock()

	// pull the actual resource from the API server.
	actualObjects, err := a.getActualObjects(ctx, gvks)
	if err != nil {
		return err
	}
	// TODO(b/165081629): Enable prune on startup (eg when cachedObjects is nil)
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
func (a *Applier) Refresh(ctx context.Context) status.MultiError {
	a.mux.Lock()
	defer a.mux.Unlock()

	gvks := make(map[schema.GroupVersionKind]bool)
	for _, resource := range a.cachedObjects {
		gvks[resource.GroupVersionKind()] = true
	}

	actualObjects, err := a.getActualObjects(ctx, gvks)
	if err != nil {
		return err

	}
	// Two way merge. Compare between the cached declared and the actual states to decide the create and update.
	diffs := diff.TwoWay(a.cachedObjects, actualObjects)
	if err := a.sync(ctx, diffs); err != nil {
		return err
	}
	return nil
}

// getActualObjects fetches the current resources from the API server.
func (a *Applier) getActualObjects(ctx context.Context, gvks map[schema.GroupVersionKind]bool) (map[core.ID]core.Object, status.MultiError) {
	var errs status.MultiError
	actual := make(map[core.ID]core.Object)
	for gvk := range gvks {
		resources := &unstructured.UnstructuredList{}
		resources.SetGroupVersionKind(gvk.GroupVersion().WithKind(gvk.Kind + "List"))
		if err := a.client.List(ctx, resources, a.listOptions...); err != nil {
			errs = status.Append(errs, status.APIServerErrorf(err, "failed to list %s resources", gvk.String()))
			continue
		}
		for _, res := range resources.Items {
			obj := res.DeepCopyObject().(core.Object)
			if coreID, err := core.IDOfRuntime(obj); err != nil {
				errs = status.Append(errs, err)
			} else {
				actual[coreID] = obj
			}
		}
	}
	return actual, errs
}

func (a *Applier) isRootApplier() bool {
	return a.scope == declared.RootReconciler
}
