package applier

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/diff"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/kptapplier"
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
	// cachedObjects stores the successfully-applied resources.
	// The applier uses this field to compare with newly parsed resources from git to
	// determine which previously declared resources should be deleted.
	// In cases when the applier fails to apply all the declared resources, the
	// successfully-applied resources will still be added into `cacheObjects`.
	cachedObjects map[core.ID]client.Object
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
	// This is called when the latest state of the git repo has not been fully applied.
	// It returns:
	//   1) a map of the GVKs which were successfully applied by the Applier;
	//   2) the errors encountered.
	Apply(ctx context.Context, desiredResources []client.Object) (map[schema.GroupVersionKind]struct{}, status.MultiError)
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
		cachedObjects: make(map[core.ID]client.Object),
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
		cachedObjects: make(map[core.ID]client.Object),
		client:        c,
		listOptions:   opts,
		scope:         declared.RootReconciler,
	}
	glog.V(4).Infof("Root applier is initialized and synced with the API server")
	return a
}

// applyProgress tracks how many resources were created, updated, deleted and unmanaged
type applyProgress struct {
	created   uint64
	updated   uint64
	deleted   uint64
	unmanaged uint64
}

func (p applyProgress) empty() bool {
	return p.created == 0 && p.updated == 0 && p.deleted == 0 && p.unmanaged == 0
}

func (p applyProgress) string() string {
	return fmt.Sprintf("created %d objects, deleted %d objects, updated %d objects, unmanaged %d objects",
		p.created, p.deleted, p.updated, p.unmanaged)
}

// sync actuates a list of Diff to make sure the actual resources in the API service are
// in sync with the declared resources.
func (a *Applier) sync(ctx context.Context, diffs []diff.Diff) status.MultiError {
	var progress applyProgress
	var errs status.MultiError

	// Sort diffs so that cluster-scoped resources are first.
	// Don't put these into a map before reading them out, or ordering will not be
	// guaranteed.
	sortByScope(diffs)
	for _, d := range diffs {
		// Take CRUD actions based on the diff between actual resource (what's stored in
		// the api server) and the declared resource (the cached git resource).
		var decl client.Object
		if d.Declared != nil {
			decl = d.Declared.DeepCopyObject().(client.Object)
		} else {
			decl = d.Actual.DeepCopyObject().(client.Object)
		}
		coreID := core.IDOf(decl)

		switch t := d.Operation(ctx, a.scope); t {
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
				progress.created++
				a.cachedObjects[coreID] = d.Declared
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
			updated, err := a.applier.Update(ctx, u, actual)
			if err != nil {
				errs = status.Append(errs, err)
			} else if updated {
				glog.V(4).Infof("updated resource %v", coreID)
				progress.updated++
				a.cachedObjects[coreID] = d.Declared
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
				progress.deleted++
				delete(a.cachedObjects, coreID)
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
				progress.unmanaged++
				a.cachedObjects[coreID] = d.Declared
			}
		case diff.ManagementConflict:
			err := kptapplier.ManagementConflictError(d.Declared)
			errs = status.Append(errs, err)
		default:
			err := status.InternalErrorf("diff type not supported: %v", t)
			errs = status.Append(errs, err)
		}
	}
	if errs == nil {
		glog.V(4).Infof("all resources are up to date.")
	}
	if progress.empty() {
		glog.V(4).Infof("The applier made no new progress")
	} else {
		glog.Infof("The applier made new progress: %s.", progress.string())
	}
	return errs
}

// Apply implements Interface.
func (a *Applier) Apply(ctx context.Context, desiredResource []client.Object) (map[schema.GroupVersionKind]struct{}, status.MultiError) {
	// create the new cache showing the new declared resource.
	newCache := make(map[core.ID]client.Object)
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
	return gvks, status.Append(errs, syncErrs)
}

// getActualObjects fetches and returns the current resources from the API
// server to match the given declared resources. It also returns a map of GVKs
// which were successfully listed by the Applier (eg not in an unknown state).
func (a *Applier) getActualObjects(ctx context.Context, declared map[core.ID]client.Object) (map[core.ID]client.Object, map[schema.GroupVersionKind]struct{}, status.MultiError) {
	gvks := make(map[schema.GroupVersionKind]struct{})
	for _, resource := range declared {
		gvks[resource.GetObjectKind().GroupVersionKind()] = struct{}{}
	}

	var errs status.MultiError
	actual := make(map[core.ID]client.Object)
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
			if decl.GetObjectKind().GroupVersionKind() == obj.GroupVersionKind() {
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
		if _, listed := gvks[obj.GetObjectKind().GroupVersionKind()]; !listed {
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
