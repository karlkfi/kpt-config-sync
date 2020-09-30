package applier

import (
	"context"
	"sort"
	"sync"

	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/diff"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
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
	// Apply updates the resource API server with the latest parsed git resource.
	// This is called when a new change in the git resource is detected. It also
	// returns a map of the GVKs which were successfully applied by the Applier.
	Apply(ctx context.Context, desiredResources []core.Object) (map[schema.GroupVersionKind]struct{}, status.MultiError)
}

var _ Interface = &Applier{}

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

	// Sort diffs so that cluster-scoped resources are first.
	// Don't put these into a map before reading them out, or ordering will not be
	// guaranteed.
	sortByScope(diffs)
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

		switch t := d.Operation(a.scope); t {
		case diff.NoOp:
			continue
		case diff.Error:
			err := nonhierarchical.IllegalManagementAnnotationError(decl,
				decl.GetAnnotations()[v1.ResourceManagementKey])
			errs = status.Append(errs, err)
		case diff.Create:
			u, err := d.UnstructuredDeclared()
			if err != nil {
				errs = status.Append(errs, err)
				continue
			}
			if _, err := a.applier.Create(ctx, u); err != nil {
				errs = status.Append(errs, err)
			} else {
				glog.V(4).Infof("created resource %v", coreID)
			}
		case diff.Update:
			u, err := d.UnstructuredDeclared()
			if err != nil {
				errs = status.Append(errs, err)
				continue
			}
			actual, err := d.UnstructuredActual()
			if err != nil {
				errs = status.Append(errs, err)
				continue
			}
			if _, err := a.applier.Update(ctx, u, actual); err != nil {
				errs = status.Append(errs, err)
			} else {
				glog.V(4).Infof("updated resource %v", coreID)
			}
		case diff.Delete:
			actual, err := d.UnstructuredActual()
			if err != nil {
				errs = status.Append(errs, err)
				continue
			}
			if _, err := a.applier.Delete(ctx, actual); err != nil {
				errs = status.Append(errs, err)
			} else {
				glog.V(4).Infof("deleted resource %v", coreID)
			}
		case diff.Unmanage:
			actual, err := d.UnstructuredActual()
			if err != nil {
				errs = status.Append(errs, err)
				continue
			}
			if _, err := a.applier.RemoveNomosMeta(ctx, actual); err != nil {
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

// Apply implements Interface.
func (a *Applier) Apply(ctx context.Context, desiredResource []core.Object) (map[schema.GroupVersionKind]struct{}, status.MultiError) {
	// create the new cache showing the new declared resource.
	newCache := make(map[core.ID]core.Object)
	for _, desired := range desiredResource {
		newCache[core.IDOf(desired)] = desired
	}

	a.mux.Lock()
	defer a.mux.Unlock()

	var errs status.MultiError
	// Pull the actual resources from the API server to compare against the
	// declared resources. Note that we do not immediately return on error here
	// because the Applier needs to try to do as much work as it can on each
	// cycle. We collect and return all errors at the end. SOme of those errors
	// are transient and resolve in future cycles based on partial work completed
	// in a previous cycle (eg ignore an error about a CR so that we can apply the
	// CRD, then a future cycle is able to apply the CR).
	// TODO(b/169717222): Here and elsewhere, pass the MultiError as a parameter.
	actualObjects, gvks, getErrs := a.getActualObjects(ctx, newCache)
	errs = status.Append(errs, getErrs)

	// TODO(b/165081629): Enable prune on startup (eg when cachedObjects is nil)
	diffs := diff.ThreeWay(newCache, a.cachedObjects, actualObjects)
	// Sync the API resource state to the git resource.
	syncErrs := a.sync(ctx, diffs)
	errs = status.Append(errs, syncErrs)

	if errs == nil {
		// Only update the cache on complete success.
		a.cachedObjects = newCache
	}
	return gvks, errs
}

// Refresh syncs and updates the API server with the (cached) git resource states.
func (a *Applier) Refresh(ctx context.Context) status.MultiError {
	a.mux.Lock()
	defer a.mux.Unlock()

	actualObjects, _, err := a.getActualObjects(ctx, a.cachedObjects)
	// Two way merge. Compare between the cached declared and the actual states to decide the create and update.
	diffs := diff.TwoWay(a.cachedObjects, actualObjects)
	// Sync the API resource state to the git resource.
	syncErrs := a.sync(ctx, diffs)

	return status.Append(err, syncErrs)
}

// getActualObjects fetches and returns the current resources from the API
// server to match the given declared resources. It also returns a map of GVKs
// which were successfully listed by the Applier (eg not in an unknown state).
func (a *Applier) getActualObjects(ctx context.Context, declared map[core.ID]core.Object) (map[core.ID]core.Object, map[schema.GroupVersionKind]struct{}, status.MultiError) {
	gvks := make(map[schema.GroupVersionKind]struct{})
	for _, resource := range declared {
		gvks[resource.GroupVersionKind()] = struct{}{}
	}

	var errs status.MultiError
	actual := make(map[core.ID]core.Object)
	for gvk := range gvks {
		resources := &unstructured.UnstructuredList{}
		resources.SetGroupVersionKind(gvk.GroupVersion().WithKind(gvk.Kind + "List"))
		if err := a.client.List(ctx, resources, a.listOptions...); err != nil {
			errs = status.Append(errs, FailedToListResources(err))
			// Remove any GVK which we were unable to list resources for. This is
			// typically caused by a new type that is not available yet. We will retry
			// again soon.
			delete(gvks, gvk)
			continue
		}

		for _, res := range resources.Items {
			obj := res.DeepCopy()
			coreID := core.IDOf(obj)
			decl, ok := declared[coreID]
			if !ok {
				continue
			}
			// It's possible for `declared` to contain resources of the same GroupKind
			// but with differing versions. In that case, those resources will show up
			// in both GVK lists. We only want to keep the resource whose GVK matches
			// the GVK of its declaration.
			if decl.GroupVersionKind() == obj.GroupVersionKind() {
				actual[coreID] = obj
			} else {
				glog.V(4).Infof("Ignoring version %q of actual resource %s.",
					obj.GroupVersionKind().Version, coreID.String())
			}
		}
	}

	// For all declared objects, mark their actual state as unknown if we were
	// unable to list it by GVK.
	for id, obj := range declared {
		if _, listed := gvks[obj.GroupVersionKind()]; !listed {
			actual[id] = diff.Unknown()
		}
	}
	return actual, gvks, errs
}

// FailedToListResourcesCode is the code that represents the Applier failing to
// list resources of a specfic GVK.
var FailedToListResourcesCode = "2007"

var failedToListResourcesBuilder = status.NewErrorBuilder(FailedToListResourcesCode)

// FailedToListResources reports that the Applier failed to list resources and
// the underlying cause.
func FailedToListResources(reason error) status.Error {
	return failedToListResourcesBuilder.Wrap(reason).
		Sprint("failed to list resources").Build()
}

func sortByScope(diffs []diff.Diff) {
	// Sort diffs so that cluster-scoped ones get added first.
	sort.Slice(diffs, func(i, j int) bool {
		return clusterScopedFirst(diffs[i], diffs[j])
	})
}

func clusterScopedFirst(left, right diff.Diff) bool {
	// Less than if left is Namespaced and right is not.
	leftNamespaced := false
	if left.Declared != nil {
		leftNamespaced = left.Declared.GetNamespace() != ""
	} else if left.Actual != nil {
		leftNamespaced = left.Actual.GetNamespace() != ""
	}

	rightNamespaced := false
	if right.Declared != nil {
		rightNamespaced = right.Declared.GetNamespace() != ""
	} else if right.Actual != nil {
		rightNamespaced = right.Actual.GetNamespace() != ""
	}

	switch {
	case leftNamespaced && rightNamespaced:
		return false
	case leftNamespaced && !rightNamespaced:
		// The only case where we want to swap left and right is if left is
		// Namespaced and right is not.
		return false
	case !leftNamespaced && rightNamespaced:
		return true
	case !leftNamespaced && !rightNamespaced:
		return false
	}

	// Unreachable code.
	return false
}
