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
	// inventory is the InventoryInfo for current Applier.
	inventory inventory.InventoryInfo
	// clientSetFunc is the function to create kpt clientSet.
	// Use this as a function so that the unit testing can mock
	// the clientSet.
	clientSetFunc func() (*clientSet, error)
	// client get and updates RepoSync and its status.
	client client.Client
	// cache stores the previously parsed git resources in memory. The applier uses this
	// cache for refresh.
	cache map[core.ID]core.Object
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
	// TODO(b/161256730): Constrains the resources due to the new labeling strategy.
	u := newInventoryUnstructured(configmanagement.ControllerNamespace, string(namespace))
	a := &Applier{
		inventory:     live.WrapInventoryInfoObj(u),
		client:        c,
		clientSetFunc: newClientSet,
		cache:         make(map[core.ID]core.Object),
		scope:         namespace,
		policy:        inventory.AdoptIfNoInventory,
	}
	glog.V(4).Infof("Applier %v is initialized", namespace)
	return a
}

// NewRootApplier initializes an applier that can fetch all resources from the API server.
func NewRootApplier(c client.Client) *Applier {
	// TODO(b/161256730): Constrains the resources due to the new labeling strategy.
	u := newInventoryUnstructured(configmanagement.ControllerNamespace, configmanagement.ControllerNamespace)
	a := &Applier{
		inventory:     live.WrapInventoryInfoObj(u),
		client:        c,
		clientSetFunc: newClientSet,
		cache:         make(map[core.ID]core.Object),
		scope:         declared.RootReconciler,
		policy:        inventory.AdoptAll,
	}
	glog.V(4).Infof("Root applier is initialized and synced with the API server")
	return a
}

// sync triggers a kpt live apply library call to apply a set of resources.
func (a *Applier) sync(ctx context.Context, objs []core.Object, cache map[core.ID]core.Object) (map[schema.GroupVersionKind]struct{}, status.MultiError) {
	var errs status.MultiError
	cs, err := a.clientSetFunc()
	if err != nil {
		return nil, ApplierError(err)
	}

	// disabledObjs are objects for which the management are disabled
	// through annotation.
	enabledObjs, disabledObjs := partitionObjs(objs)
	if len(disabledObjs) > 0 {
		err = cs.handleDisabledObjects(ctx, a.inventory, disabledObjs)
		if err != nil {
			return nil, status.Append(errs, err)
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
		case event.ApplyType:
			// ApplyEvent.Type has two types: ApplyEventResourceUpdate and ApplyEventCompleted.
			// ApplyEventResourceUpdate is for applying a single resource;
			// ApplyEventCompleted indicates all resources have been applied.
			if e.ApplyEvent.Type == event.ApplyEventResourceUpdate {
				id := idFrom(e.ApplyEvent.Identifier)
				if e.ApplyEvent.Error != nil {
					switch e.ApplyEvent.Error.(type) {
					case *applyerror.UnknownTypeError:
						errs = status.Append(errs, ApplierError(e.ApplyEvent.Error))
						unknownTypeResources[id] = struct{}{}
					case *inventory.InventoryOverlapError:
						errs = status.Append(errs, ManagementConflictError(cache[id]))
					default:
						// The default case covers other reason for failed applying a resource.
						errs = status.Append(errs, ApplierError(e.ApplyEvent.Error))
					}
				} else {
					glog.V(4).Infof("applied resource %v", id)
				}
			}
		case event.PruneType:
			if e.PruneEvent.Error != nil {
				errs = status.Append(errs, ApplierError(e.PruneEvent.Error))
			} else if glog.V(4) {
				id := idFrom(e.PruneEvent.Identifier)
				if e.PruneEvent.Operation == event.PruneSkipped {
					glog.Infof("skipped pruning resource %v", id)
				} else {
					glog.Infof("pruned resource %v", id)
				}
			}
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

	glog.V(4).Infof("all resources are up to date.")
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
	gvks, errs := a.sync(ctx, desiredResource, newCache)
	if errs == nil {
		// Only update the cache on complete success.
		a.cache = newCache
	}
	return gvks, errs
}

// Refresh syncs and updates the API server with the (cached) git resource states.
func (a *Applier) Refresh(ctx context.Context) status.MultiError {
	a.mux.Lock()
	defer a.mux.Unlock()

	objs := []core.Object{}
	for _, obj := range a.cache {
		objs = append(objs, obj)
	}
	_, errs := a.sync(ctx, objs, a.cache)
	return errs
}

// newInventoryUnstructured creates an inventory object as an unstructured.
func newInventoryUnstructured(namespace, name string) *unstructured.Unstructured {
	id := namespace + "_" + name
	u := live.ResourceGroupUnstructured(name, namespace, id)
	labels := u.GetLabels()
	labels[v1.ManagedByKey] = v1.ManagedByValue
	u.SetLabels(labels)
	return u
}
