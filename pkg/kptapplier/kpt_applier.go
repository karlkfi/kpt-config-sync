package kptapplier

import (
	"context"
	"sync"

	"github.com/GoogleContainerTools/kpt/pkg/live"
	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configmanagement"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/cli-utils/pkg/apply"
	applyerror "sigs.k8s.io/cli-utils/pkg/apply/error"
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Applier declares the Applier component in the Multi Repo Reconciler Process.
type Applier struct {
	// inventory policy for the applier.
	policy inventory.InventoryPolicy
	// inventory is the inventory ResourceGroup for current Applier.
	inventory *live.InventoryResourceGroup
	// clientSetFunc is the function to create kpt clientSet.
	// Use this as a function so that the unit testing can mock
	// the clientSet.
	clientSetFunc func() (*clientSet, error)
	// client get and updates RepoSync and its status.
	client client.Client
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
func NewNamespaceApplier(c client.Client, namespace declared.Scope) *Applier {
	u := newInventoryUnstructured(string(namespace))
	a := &Applier{
		inventory:     live.WrapInventoryResourceGroup(u),
		client:        c,
		clientSetFunc: newClientSet,
		scope:         namespace,
		policy:        inventory.AdoptIfNoInventory,
	}
	glog.V(4).Infof("Applier %v is initialized", namespace)
	return a
}

// NewRootApplier initializes an applier that can fetch all resources from the API server.
func NewRootApplier(c client.Client) *Applier {
	u := newInventoryUnstructured(configmanagement.ControllerNamespace)
	a := &Applier{
		inventory:     live.WrapInventoryResourceGroup(u),
		client:        c,
		clientSetFunc: newClientSet,
		scope:         declared.RootReconciler,
		policy:        inventory.AdoptAll,
	}
	glog.V(4).Infof("Root applier is initialized and synced with the API server")
	return a
}

func processApplyEvent(e event.ApplyEvent, stats *applyEventStats, cache map[core.ID]core.Object, unknownTypeResources map[core.ID]struct{}) status.Error {
	// ApplyEvent.Type has two types: ApplyEventResourceUpdate and ApplyEventCompleted.
	// ApplyEventResourceUpdate is for applying a single resource;
	// ApplyEventCompleted indicates all resources have been applied.
	if e.Type != event.ApplyEventResourceUpdate {
		return nil
	}

	id := idFrom(e.Identifier)
	if e.Error != nil {
		stats.errCount++
		switch e.Error.(type) {
		case *applyerror.UnknownTypeError:
			unknownTypeResources[id] = struct{}{}
			return ApplierError(e.Error)
		case *inventory.InventoryOverlapError:
			return ManagementConflictError(cache[id])
		default:
			// The default case covers other reason for failed applying a resource.
			return ApplierError(e.Error)
		}
	}

	if e.Operation == event.Unchanged {
		glog.V(7).Infof("applied [op: %v] resource %v", e.Operation, id)
	} else {
		glog.V(4).Infof("applied [op: %v] resource %v", e.Operation, id)
		stats.eventByOp[e.Operation]++
	}
	return nil
}

func processPruneEvent(e event.PruneEvent, stats *pruneEventStats) status.Error {
	if e.Error != nil {
		stats.errCount++
		return ApplierError(e.Error)
	}

	id := idFrom(e.Identifier)
	if e.Operation == event.PruneSkipped {
		// A PruneSkipped event indicates no change on any cluster object.
		glog.V(4).Infof("skipped pruning resource %v", id)
	} else {
		glog.V(4).Infof("pruned resource %v", id)
		stats.eventByOp[e.Operation]++
	}
	return nil
}

// sync triggers a kpt live apply library call to apply a set of resources.
func (a *Applier) sync(ctx context.Context, objs []core.Object, cache map[core.ID]core.Object) (map[schema.GroupVersionKind]struct{}, status.MultiError) {
	var errs status.MultiError
	cs, err := a.clientSetFunc()
	if err != nil {
		return nil, ApplierError(err)
	}

	stats := newApplyStats()
	// disabledObjs are objects for which the management are disabled
	// through annotation.
	enabledObjs, disabledObjs := partitionObjs(objs)
	if len(disabledObjs) > 0 {
		disabledCount, err := cs.handleDisabledObjects(ctx, a.inventory, disabledObjs)
		if err != nil {
			return nil, status.Append(errs, err)
		}
		stats.disableObjs = disabledObjStats{
			total:     uint64(len(disabledObjs)),
			succeeded: disabledCount,
		}
	}

	resources, toUnsErrs := toUnstructured(enabledObjs)
	if toUnsErrs != nil {
		return nil, toUnsErrs
	}

	unknownTypeResources := make(map[core.ID]struct{})

	events := cs.apply(ctx, a.inventory, resources, apply.Options{InventoryPolicy: a.policy})
	for e := range events {
		switch e.Type {
		case event.ErrorType:
			errs = status.Append(errs, ApplierError(e.ErrorEvent.Err))
			stats.errorTypeEvents++
		case event.ApplyType:
			errs = status.Append(errs, processApplyEvent(e.ApplyEvent, &stats.applyEvent, cache, unknownTypeResources))
		case event.PruneType:
			errs = status.Append(errs, processPruneEvent(e.PruneEvent, &stats.pruneEvent))
		default:
			glog.V(4).Infof("skipped %v event", e.Type)
		}
	}

	gvks := make(map[schema.GroupVersionKind]struct{})
	for _, resource := range objs {
		id := core.IDOf(resource)
		if _, found := unknownTypeResources[id]; found {
			continue
		}
		gvks[resource.GroupVersionKind()] = struct{}{}
	}
	if errs == nil {
		glog.V(4).Infof("all resources are up to date.")
	}

	if stats.empty() {
		glog.V(4).Infof("The applier made no new progress")
	} else {
		glog.Infof("The applier made new progress: %s.", stats.string())
	}
	return gvks, errs
}

// Apply implements Interface.
func (a *Applier) Apply(ctx context.Context, desiredResource []core.Object) (map[schema.GroupVersionKind]struct{}, status.MultiError) {
	// Create the new cache showing the new declared resource.
	newCache := make(map[core.ID]core.Object)
	for _, desired := range desiredResource {
		newCache[core.IDOf(desired)] = desired
	}

	a.mux.Lock()
	defer a.mux.Unlock()

	// Pull the actual resources from the API server to compare against the
	// declared resources. Note that we do not immediately return on error here
	// because the Applier needs to try to do as much work as it can on each
	// cycle. We collect and return all errors at the end. Some of those errors
	// are transient and resolve in future cycles based on partial work completed
	// in a previous cycle (eg ignore an error about a CR so that we can apply the
	// CRD, then a future cycle is able to apply the CR).
	// TODO(b/169717222): Here and elsewhere, pass the MultiError as a parameter.
	return a.sync(ctx, desiredResource, newCache)
}

// newInventoryUnstructured creates an inventory object as an unstructured.
func newInventoryUnstructured(name string) *unstructured.Unstructured {
	id := InventoryID(name)
	u := live.ResourceGroupUnstructured(name, configmanagement.ControllerNamespace, id)
	labels := u.GetLabels()
	labels[v1.ManagedByKey] = v1.ManagedByValue
	u.SetLabels(labels)
	return u
}

// InventoryID returns the inventory id of an inventory object.
// The inventory objects generated by ConfigSync are all in the config-management-system namespace.
// The name of the inventory object is assigned to the namespace for the current applier.
// So that the name and namespace of the inventory objects are uniquely determined by a RootSync/RepoSync CR.
// They don't collide with each other or with inventory objects from other namespaces.
// The inventory ID is assigned as <NAMESPACE>_<NAME>.
func InventoryID(name string) string {
	return configmanagement.ControllerNamespace + "_" + name
}
