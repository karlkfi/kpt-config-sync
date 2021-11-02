package applier

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/GoogleContainerTools/kpt/pkg/live"
	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/metadata"
	m "github.com/google/nomos/pkg/metrics"
	"github.com/google/nomos/pkg/resourcegroup"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/differ"
	"github.com/google/nomos/pkg/syncer/metrics"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/cli-utils/pkg/apply"
	applyerror "sigs.k8s.io/cli-utils/pkg/apply/error"
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	"sigs.k8s.io/cli-utils/pkg/common"
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
	clientSetFunc func(client.Client) (*clientSet, error)
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
	Apply(ctx context.Context, desiredResources []client.Object) (map[schema.GroupVersionKind]struct{}, status.MultiError)
}

var _ Interface = &Applier{}

// NewNamespaceApplier initializes an applier that fetches a certain namespace's resources from
// the API server.
func NewNamespaceApplier(c client.Client, namespace declared.Scope) (*Applier, error) {
	u := newInventoryUnstructured(configsync.RepoSyncName, string(namespace))
	inv, ok := live.WrapInventoryObj(u).(*live.InventoryResourceGroup)
	if !ok {
		return nil, errors.New("failed to create an ResourceGroup object")
	}

	a := &Applier{
		inventory:     inv,
		client:        c,
		clientSetFunc: newClientSet,
		scope:         namespace,
		policy:        inventory.AdoptIfNoInventory,
	}
	glog.V(4).Infof("Applier %v is initialized", namespace)
	return a, nil
}

// NewRootApplier initializes an applier that can fetch all resources from the API server.
func NewRootApplier(c client.Client) (*Applier, error) {
	u := newInventoryUnstructured(configsync.RootSyncName, configmanagement.ControllerNamespace)
	inv, ok := live.WrapInventoryObj(u).(*live.InventoryResourceGroup)
	if !ok {
		return nil, errors.New("failed to create an ResourceGroup object")
	}

	a := &Applier{
		inventory:     inv,
		client:        c,
		clientSetFunc: newClientSet,
		scope:         declared.RootReconciler,
		policy:        inventory.AdoptAll,
	}
	glog.V(4).Infof("Root applier is initialized and synced with the API server")
	return a, nil
}

func processApplyEvent(ctx context.Context, e event.ApplyEvent, stats *applyEventStats, cache map[core.ID]client.Object, unknownTypeResources map[core.ID]struct{}) status.Error {
	id := idFrom(e.Identifier)
	if e.Error != nil {
		stats.errCount++
		switch e.Error.(type) {
		case *applyerror.UnknownTypeError:
			unknownTypeResources[id] = struct{}{}
			return ErrorForResource(e.Error, id)
		case *inventory.InventoryOverlapError:
			return ManagementConflictError(cache[id])
		default:
			// The default case covers other reason for failed applying a resource.
			return ErrorForResource(e.Error, id)
		}
	}

	if e.Operation == event.Unchanged {
		glog.V(7).Infof("applied [op: %v] resource %v", e.Operation, id)
	} else {
		glog.V(4).Infof("applied [op: %v] resource %v", e.Operation, id)
		handleMetrics(ctx, "update", e.Error, id.WithVersion(""))
		stats.eventByOp[e.Operation]++
	}
	return nil
}

func processPruneEvent(ctx context.Context, e event.PruneEvent, stats *pruneEventStats, cs *clientSet) status.Error {
	if e.Error != nil {
		id := idFrom(e.Identifier)
		stats.errCount++
		return ErrorForResource(e.Error, id)
	}

	id := idFrom(e.Identifier)
	if e.Operation == event.PruneSkipped {
		glog.V(4).Infof("skipped pruning resource %v", id)
		if e.Object != nil && e.Object.GetObjectKind().GroupVersionKind().GroupKind() == kinds.Namespace().GroupKind() && differ.SpecialNamespaces[e.Object.GetName()] {
			// the `client.lifecycle.config.k8s.io/deletion: detach` annotation is not a part of the Config Sync metadata, and will not be removed here.
			err := cs.disableObject(ctx, e.Object)
			if err != nil {
				errorMsg := "failed to remove the Config Sync metadata from %v (which is a special namespace): %v"
				glog.Errorf(errorMsg, id, err)
				return applierErrorBuilder.Wrap(fmt.Errorf(errorMsg, id, err)).Build()
			}
			glog.V(4).Infof("removed the Config Sync metadata from %v (which is a special namespace)", id)
		}
	} else {
		glog.V(4).Infof("pruned resource %v", id)
		handleMetrics(ctx, "delete", e.Error, id.WithVersion(""))
		stats.eventByOp[e.Operation]++
	}
	return nil
}

func handleMetrics(ctx context.Context, operation string, err error, gvk schema.GroupVersionKind) {
	// TODO(b/180744881) capture the apply duration in the kpt apply library.
	start := time.Now()

	m.RecordAPICallDuration(ctx, operation, m.StatusTagKey(err), gvk, start)
	metrics.Operations.WithLabelValues(operation, gvk.Kind, metrics.StatusLabel(err)).Inc()
	m.RecordApplyOperation(ctx, operation, m.StatusTagKey(err), gvk)
}

// sync triggers a kpt live apply library call to apply a set of resources.
func (a *Applier) sync(ctx context.Context, objs []client.Object, cache map[core.ID]client.Object) (map[schema.GroupVersionKind]struct{}, status.MultiError) {
	var errs status.MultiError
	cs, err := a.clientSetFunc(a.client)
	if err != nil {
		return nil, Error(err)
	}

	stats := newApplyStats()
	// disabledObjs are objects for which the management are disabled
	// through annotation.
	enabledObjs, disabledObjs := partitionObjs(objs)
	if len(disabledObjs) > 0 {
		glog.Infof("%v objects to be disabled: %v", len(disabledObjs), core.GKNNs(disabledObjs))
		disabledCount, err := cs.handleDisabledObjects(ctx, a.inventory, disabledObjs)
		if err != nil {
			return nil, status.Append(errs, err)
		}
		stats.disableObjs = disabledObjStats{
			total:     uint64(len(disabledObjs)),
			succeeded: disabledCount,
		}
	}
	glog.Infof("%v objects to be applied: %v", len(enabledObjs), core.GKNNs(enabledObjs))
	resources, toUnsErrs := toUnstructured(enabledObjs)
	if toUnsErrs != nil {
		return nil, toUnsErrs
	}

	unknownTypeResources := make(map[core.ID]struct{})
	options := apply.Options{
		ServerSideOptions: common.ServerSideOptions{
			ServerSideApply: true,
			ForceConflicts:  true,
			FieldManager:    configsync.FieldManager,
		},
		InventoryPolicy: a.policy,
	}

	events := cs.apply(ctx, a.inventory, resources, options)
	for e := range events {
		switch e.Type {
		case event.ErrorType:
			errs = status.Append(errs, Error(e.ErrorEvent.Err))
			stats.errorTypeEvents++
		case event.WaitType:
			glog.Info(e.WaitEvent.Error)
		case event.ApplyType:
			errs = status.Append(errs, processApplyEvent(ctx, e.ApplyEvent, &stats.applyEvent, cache, unknownTypeResources))
		case event.PruneType:
			errs = status.Append(errs, processPruneEvent(ctx, e.PruneEvent, &stats.pruneEvent, cs))
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
		gvks[resource.GetObjectKind().GroupVersionKind()] = struct{}{}
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
func (a *Applier) Apply(ctx context.Context, desiredResource []client.Object) (map[schema.GroupVersionKind]struct{}, status.MultiError) {
	// Create the new cache showing the new declared resource.
	newCache := make(map[core.ID]client.Object)
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
func newInventoryUnstructured(name, namespace string) *unstructured.Unstructured {
	id := InventoryID(namespace)
	u := resourcegroup.Unstructured(name, namespace, id)
	core.SetLabel(u, metadata.ManagedByKey, metadata.ManagedByValue)
	core.SetAnnotation(u, metadata.ResourceManagementKey, metadata.ResourceManagementEnabled)
	return u
}

// InventoryID returns the inventory id of an inventory object.
// The inventory object generated by ConfigSync is in the same namespace as RootSync or RepoSync.
// The name of the inventory object is assigned to "repo-sync" for namespaced reconciler
// or "root-sync" for the root reconciler.
// So that the name and namespace of the inventory objects are uniquely determined by a RootSync/RepoSync CR.
// The inventory ID is assigned as <NAMESPACE>_<NAME>.
func InventoryID(namespace string) string {
	var name string
	if namespace == configmanagement.ControllerNamespace {
		name = configsync.RootSyncName
	} else {
		name = configsync.RepoSyncName
	}
	return namespace + "_" + name
}
