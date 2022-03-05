// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package applier

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/GoogleContainerTools/kpt/pkg/live"
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
	"github.com/google/nomos/pkg/util"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
	"sigs.k8s.io/cli-utils/pkg/apply"
	applyerror "sigs.k8s.io/cli-utils/pkg/apply/error"
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/object/dependson"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// maxRequestBytesStr defines the max request bytes on the etcd server.
	// It is defined in https://github.com/etcd-io/etcd/blob/release-3.4/embed/config.go#L56
	maxRequestBytesStr = "1.5M"

	// maxRequestBytes defines the max request bytes on the etcd server.
	// It is defined in https://github.com/etcd-io/etcd/blob/release-3.4/embed/config.go#L56
	maxRequestBytes = int64(1.5 * 1024 * 1024)
)

// Applier declares the Applier component in the Multi Repo Reconciler Process.
type Applier struct {
	// inventory policy for the applier.
	policy inventory.Policy
	// inventory is the inventory ResourceGroup for current Applier.
	inventory *live.InventoryResourceGroup
	// clientSetFunc is the function to create kpt clientSet.
	// Use this as a function so that the unit testing can mock
	// the clientSet.
	clientSetFunc func(client.Client) (*clientSet, error)
	// client get and updates RepoSync and its status.
	client client.Client
	// errs tracks all the errors the applier encounters.
	// This field is cleared at the start of the `Applier.Apply` method
	errs status.MultiError
	// syncing indicates whether the applier is syncing.
	syncing bool
	// name and namespace of the RootSync|RepoSync object
	// for the current applier.
	syncName      string
	syncNamespace string
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
	// Errors returns the errors encountered during apply.
	Errors() status.MultiError
	// Syncing indicates whether the applier is syncing.
	Syncing() bool
}

var _ Interface = &Applier{}

// NewNamespaceApplier initializes an applier that fetches a certain namespace's resources from
// the API server.
func NewNamespaceApplier(c client.Client, namespace declared.Scope, syncName string) (*Applier, error) {
	u := newInventoryUnstructured(syncName, string(namespace))
	inv, ok := live.WrapInventoryObj(u).(*live.InventoryResourceGroup)
	if !ok {
		return nil, errors.New("failed to create an ResourceGroup object")
	}

	a := &Applier{
		inventory:     inv,
		client:        c,
		clientSetFunc: newClientSet,
		policy:        inventory.PolicyAdoptIfNoInventory,
		syncName:      syncName,
		syncNamespace: string(namespace),
	}
	klog.V(4).Infof("Applier %s/%s is initialized", namespace, syncName)
	return a, nil
}

// NewRootApplier initializes an applier that can fetch all resources from the API server.
func NewRootApplier(c client.Client, syncName string) (*Applier, error) {
	u := newInventoryUnstructured(syncName, configmanagement.ControllerNamespace)
	inv, ok := live.WrapInventoryObj(u).(*live.InventoryResourceGroup)
	if !ok {
		return nil, errors.New("failed to create an ResourceGroup object")
	}

	a := &Applier{
		inventory:     inv,
		client:        c,
		clientSetFunc: newClientSet,
		policy:        inventory.PolicyAdoptAll,
	}
	klog.V(4).Infof("Root applier %s is initialized and synced with the API server", syncName)
	return a, nil
}

func processApplyEvent(ctx context.Context, e event.ApplyEvent, stats *applyEventStats, objsReconciled map[core.ID]struct{}, cache map[core.ID]client.Object, unknownTypeResources map[core.ID]struct{}) status.Error {
	id := idFrom(e.Identifier)
	if e.Error != nil {
		stats.errCount++
		switch e.Error.(type) {
		case *applyerror.UnknownTypeError:
			unknownTypeResources[id] = struct{}{}
			return ErrorForResource(e.Error, id)
		case *inventory.InventoryOverlapError:
			// TODO: return ManagementConflictError with the conflicting manager if
			// cli-utils supports reporting the conflicting manager in InventoryOverlapError.
			return KptManagementConflictError(cache[id])
		default:
			// The default case covers other reason for failed applying a resource.
			return ErrorForResource(e.Error, id)
		}
	}

	if e.Operation == event.Unchanged {
		if err := handleSkipEvent(e.Resource, id, objsReconciled); err != nil {
			stats.errCount++
			return err
		}
		klog.V(7).Infof("applied [op: %v] resource %v", e.Operation, id)
	} else {
		klog.V(4).Infof("applied [op: %v] resource %v", e.Operation, id)
		handleMetrics(ctx, "update", e.Error, id.WithVersion(""))
		stats.eventByOp[e.Operation]++
	}
	return nil
}

func processWaitEvent(e event.WaitEvent, objReconciled map[core.ID]struct{}) {
	id := idFrom(e.Identifier)
	if e.Operation == event.Reconciled {
		objReconciled[id] = struct{}{}
	}
}

// handleSkipEvent translates from skipped event into resource error.
// It is only used to catch the skipped apply/prune operation due to dependencies are not ready.
func handleSkipEvent(obj *unstructured.Unstructured, id core.ID, objsReconciled map[core.ID]struct{}) status.Error {
	if obj == nil {
		return nil
	}
	dependsOnStr := core.GetAnnotation(obj, dependson.Annotation)
	if dependsOnStr == "" {
		return nil
	}

	deps, err := dependson.ParseDependencySet(dependsOnStr)
	if err != nil {
		return ErrorForResource(err, id)
	}

	unReconciled := []core.ID{}
	for _, dep := range deps {
		if _, found := objsReconciled[idFrom(dep)]; !found {
			unReconciled = append(unReconciled, idFrom(dep))
		}
	}
	if len(unReconciled) > 0 {
		klog.Errorf("dependencies of %v are not ready: %v", id, unReconciled)
		return ErrorForResource(fmt.Errorf("dependencies are not reconciled: %v", unReconciled), id)
	}
	return nil
}

func processPruneEvent(ctx context.Context, e event.PruneEvent, stats *pruneEventStats, objsReconciled map[core.ID]struct{}, cs *clientSet) status.Error {
	if e.Error != nil {
		id := idFrom(e.Identifier)
		stats.errCount++
		return ErrorForResource(e.Error, id)
	}

	id := idFrom(e.Identifier)
	if e.Operation == event.PruneSkipped {
		klog.V(4).Infof("skipped pruning resource %v", id)
		if e.Object != nil && e.Object.GetObjectKind().GroupVersionKind().GroupKind() == kinds.Namespace().GroupKind() && differ.SpecialNamespaces[e.Object.GetName()] {
			// the `client.lifecycle.config.k8s.io/deletion: detach` annotation is not a part of the Config Sync metadata, and will not be removed here.
			err := cs.disableObject(ctx, e.Object)
			if err != nil {
				errorMsg := "failed to remove the Config Sync metadata from %v (which is a special namespace): %v"
				klog.Errorf(errorMsg, id, err)
				return applierErrorBuilder.Wrap(fmt.Errorf(errorMsg, id, err)).Build()
			}
			klog.V(4).Infof("removed the Config Sync metadata from %v (which is a special namespace)", id)
		}
		// TODO: handle the skip prune events due to dependency
	} else {
		klog.V(4).Infof("pruned resource %v", id)
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

// checkInventoryObjectSize checks the inventory object size limit.
// If it is close to the size limit 1M, log a warning.
func (a *Applier) checkInventoryObjectSize(ctx context.Context, c client.Client) {
	u := newInventoryUnstructured(a.syncName, a.syncNamespace)
	err := c.Get(ctx, client.ObjectKey{Namespace: a.syncNamespace, Name: a.syncName}, u)
	if err == nil {
		size, err := getObjectSize(u)
		if err != nil {
			klog.Warningf("Failed to marshal ResourceGroup %s/%s to get its size: %s", a.syncNamespace, a.syncName, err)
		}
		if int64(size) > maxRequestBytes/2 {
			klog.Warningf("ResourceGroup %s/%s is close to the maximum object size limit (size: %d, max: %s). "+
				"There are too many resources being synced than Config Sync can handle! Please split your repo into smaller repos "+
				"to avoid future failure.", a.syncNamespace, a.syncName, size, maxRequestBytesStr)
		}
	}
}

// sync triggers a kpt live apply library call to apply a set of resources.
func (a *Applier) sync(ctx context.Context, objs []client.Object, cache map[core.ID]client.Object) (map[schema.GroupVersionKind]struct{}, status.MultiError) {
	cs, err := a.clientSetFunc(a.client)
	if err != nil {
		return nil, Error(err)
	}
	a.checkInventoryObjectSize(ctx, cs.client)

	stats := newApplyStats()
	// disabledObjs are objects for which the management are disabled
	// through annotation.
	enabledObjs, disabledObjs := partitionObjs(objs)
	if len(disabledObjs) > 0 {
		klog.Infof("%v objects to be disabled: %v", len(disabledObjs), core.GKNNs(disabledObjs))
		disabledCount, err := cs.handleDisabledObjects(ctx, a.inventory, disabledObjs)
		if err != nil {
			a.errs = status.Append(a.errs, err)
			return nil, a.errs
		}
		stats.disableObjs = disabledObjStats{
			total:     uint64(len(disabledObjs)),
			succeeded: disabledCount,
		}
	}
	klog.Infof("%v objects to be applied: %v", len(enabledObjs), core.GKNNs(enabledObjs))
	resources, toUnsErrs := toUnstructured(enabledObjs)
	if toUnsErrs != nil {
		return nil, toUnsErrs
	}

	unknownTypeResources := make(map[core.ID]struct{})
	options := apply.ApplierOptions{
		ServerSideOptions: common.ServerSideOptions{
			ServerSideApply: true,
			ForceConflicts:  true,
			FieldManager:    configsync.FieldManager,
		},
		InventoryPolicy: a.policy,
		// Leaving ReconcileTimeout and PruneTimeout unset may cause a WaitTask to wait forever.
		// ReconcileTimeout defines the timeout for a wait task after an apply task.
		// ReconcileTimeout is a task-level setting instead of an object-level setting.
		ReconcileTimeout: time.Minute,
		// PruneTimeout defines the timeout for a wait task after a prune task.
		// PruneTimeout is a task-level setting instead of an object-level setting.
		PruneTimeout: time.Minute,
	}

	events := cs.apply(ctx, a.inventory, resources, options)
	for e := range events {
		switch e.Type {
		case event.InitType:
			for _, ag := range e.InitEvent.ActionGroups {
				klog.Info("InitEvent", ag)
			}
		case event.ActionGroupType:
			klog.Info(e.ActionGroupEvent)
		case event.ErrorType:
			if util.IsRequestTooLargeError(e.ErrorEvent.Err) {
				a.errs = status.Append(a.errs, largeResourceGroupError(e.ErrorEvent.Err, idFromInventory(a.inventory)))
			} else {
				a.errs = status.Append(a.errs, Error(e.ErrorEvent.Err))
			}
			stats.errorTypeEvents++
		case event.WaitType:
			// Log WaitEvent at the verbose level of 4 due to the number of WaitEvent.
			// For every object which is skipped to apply/prune, there will be one ReconcileSkipped WaitEvent.
			// For every object which is not skipped to apply/prune, there will be at least two WaitEvent:
			// one ReconcilePending WaitEvent and one Reconciled/ReconcileFailed/ReconcileTimeout WaitEvent. In addition,
			// a reconciled object may become pending before a wait task times out.
			// Record the objs that have been reconciled.
			klog.V(4).Info(e.WaitEvent)
			processWaitEvent(e.WaitEvent, stats.objsReconciled)
		case event.ApplyType:
			a.errs = status.Append(a.errs, processApplyEvent(ctx, e.ApplyEvent, &stats.applyEvent, stats.objsReconciled, cache, unknownTypeResources))
		case event.PruneType:
			a.errs = status.Append(a.errs, processPruneEvent(ctx, e.PruneEvent, &stats.pruneEvent, stats.objsReconciled, cs))
		default:
			klog.V(4).Infof("skipped %v event", e.Type)
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
	if a.errs == nil {
		klog.V(4).Infof("all resources are up to date.")
	}

	if stats.empty() {
		klog.V(4).Infof("The applier made no new progress")
	} else {
		klog.Infof("The applier made new progress: %s.", stats.string())
	}
	return gvks, a.errs
}

// Errors implements Interface.
// Errors returns the errors encountered during apply.
func (a *Applier) Errors() status.MultiError {
	return a.errs
}

// Syncing implements Interface.
// Syncing returns whether the applier is syncing.
func (a *Applier) Syncing() bool {
	return a.syncing
}

// Apply implements Interface.
func (a *Applier) Apply(ctx context.Context, desiredResource []client.Object) (map[schema.GroupVersionKind]struct{}, status.MultiError) {
	// Clear the `errs` field at the start.
	a.errs = nil
	// Set the `syncing` field to `true` at the start.
	a.syncing = true

	defer func() {
		// Make sure to clear the `syncing` field before `Apply` returns.
		a.syncing = false
	}()

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
	// TODO: Here and elsewhere, pass the MultiError as a parameter.
	return a.sync(ctx, desiredResource, newCache)
}

// newInventoryUnstructured creates an inventory object as an unstructured.
func newInventoryUnstructured(name, namespace string) *unstructured.Unstructured {
	id := InventoryID(name, namespace)
	u := resourcegroup.Unstructured(name, namespace, id)
	core.SetLabel(u, metadata.ManagedByKey, metadata.ManagedByValue)
	core.SetLabel(u, metadata.SyncNamespaceLabel, namespace)
	core.SetLabel(u, metadata.SyncNameLabel, name)
	core.SetAnnotation(u, metadata.ResourceManagementKey, metadata.ResourceManagementEnabled)
	return u
}

// InventoryID returns the inventory id of an inventory object.
// The inventory object generated by ConfigSync is in the same namespace as RootSync or RepoSync.
// The inventory ID is assigned as <NAMESPACE>_<NAME>.
func InventoryID(name, namespace string) string {
	return namespace + "_" + name
}
