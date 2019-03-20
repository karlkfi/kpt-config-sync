/*
Copyright 2018 The CSP Config Management Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package reconcile

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/cache"
	"github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/decode"
	"github.com/google/nomos/pkg/syncer/differ"
	"github.com/google/nomos/pkg/syncer/labeling"
	"github.com/google/nomos/pkg/syncer/metrics"
	"github.com/google/nomos/pkg/util/namespaceutil"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const reconcileTimeout = time.Minute * 5

var (
	_ reconcile.Reconciler = &NamespaceConfigReconciler{}

	// reservedNamespaceConfig is a dummy namespace config used to represent
	// the policy content of non-removable namespaces (like "default") when the
	// corresponding NamespaceConfig is deleted from the repo.  Instead of
	// deleting the namespace and its resources, we apply a change that removes
	// all managed resources from the namespace, but does not attempt to delete
	// the namespace.
	// TODO(filmil): See if there is an easy way to hide this nil object.
	reservedNamespaceConfig = &v1.NamespaceConfig{}
)

// NamespaceConfigReconciler reconciles a NamespaceConfig object.
type NamespaceConfigReconciler struct {
	client   *client.Client
	applier  Applier
	cache    cache.GenericCache
	recorder record.EventRecorder
	decoder  decode.Decoder
	toSync   []schema.GroupVersionKind
	// A cancelable ambient context for all reconciler operations.
	ctx context.Context
}

// NewNamespaceConfigReconciler returns a new NamespaceConfigReconciler.
func NewNamespaceConfigReconciler(ctx context.Context, client *client.Client, applier Applier, cache cache.GenericCache, recorder record.EventRecorder,
	decoder decode.Decoder, toSync []schema.GroupVersionKind) *NamespaceConfigReconciler {
	return &NamespaceConfigReconciler{
		client:   client,
		applier:  applier,
		cache:    cache,
		recorder: recorder,
		decoder:  decoder,
		toSync:   toSync,
		ctx:      ctx,
	}
}

// Reconcile is the Reconcile callback for NamespaceConfigReconciler.
func (r *NamespaceConfigReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	metrics.EventTimes.WithLabelValues("reconcileNamespaceConfig").Set(float64(now().Unix()))
	reconcileTimer := prometheus.NewTimer(
		metrics.NamespaceReconcileDuration.WithLabelValues(request.Name))
	defer reconcileTimer.ObserveDuration()

	ctx, cancel := context.WithTimeout(r.ctx, reconcileTimeout)
	defer cancel()

	name := request.Name
	glog.Infof("Reconciling NamespaceConfig: %q", name)
	if namespaceutil.IsReserved(name) {
		glog.Errorf("Trying to reconcile a NamespaceConfig corresponding to a reserved namespace: %q", name)
		// We don't return an error, because we should never be reconciling these NamespaceConfigs in the first place.
		return reconcile.Result{}, nil
	}

	err := r.reconcileNamespaceConfig(ctx, name)
	if err != nil {
		glog.Errorf("Could not reconcile namespaceconfig %q: %v", name, err)
	}
	return reconcile.Result{}, err
}

// namespaceConfigState enumerates possible states for NamespaceConfigs
type namespaceConfigState string

const (
	namespaceConfigStateNotFound  = namespaceConfigState("notFound")  // the policy node does not exist
	namespaceConfigStateNamespace = namespaceConfigState("namespace") // the policy node is declared as a namespace
)

// getNamespaceConfigState normalizes the state of the policy node and returns the node.
func (r *NamespaceConfigReconciler) getNamespaceConfigState(
	ctx context.Context,
	name string,
) (namespaceConfigState, *v1.NamespaceConfig) {
	node := &v1.NamespaceConfig{}
	err := r.cache.Get(ctx, apitypes.NamespacedName{Name: name}, node)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return namespaceConfigStateNotFound, nil
		}
		panic(errors.Wrap(err, "cache returned error other than not found, this should not happen"))
	}
	node.SetGroupVersionKind(kinds.NamespaceConfig())

	return namespaceConfigStateNamespace, node
}

// namespaceState enumerates possible states for the namespace
type namespaceState string

const (
	namespaceStateNotFound  = namespaceState("notFound")   // The namespace does not exist.
	namespaceStateUnset     = namespaceState("exists")     // The namespace exists but the management label is unset.
	namespaceStateManaged   = namespaceState("manageFull") // Management is enabled for the namespace.
	namespaceStateUnmanaged = namespaceState("disabled")   // Management is disabled for the Namespace.
)

// getNamespaceState normalizes the state of the namespace and retrieves the current value.
func (r *NamespaceConfigReconciler) getNamespaceState(
	ctx context.Context,
	name string,
	syncErrs *[]v1.NamespaceConfigSyncError,
) (namespaceState, *corev1.Namespace, error) {
	ns := &corev1.Namespace{}
	err := r.cache.Get(ctx, apitypes.NamespacedName{Name: name}, ns)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return namespaceStateNotFound, nil, nil
		}
		return namespaceStateNotFound, nil, errors.Wrapf(err, "got unhandled lister error")
	}

	value, found := ns.Annotations[v1.ResourceManagementKey]
	switch {
	case !found:
		return namespaceStateUnset, ns, nil
	case value == v1.ResourceManagementEnabled:
		return namespaceStateManaged, ns, nil
	case value == v1.ResourceManagementDisabled:
		return namespaceStateUnmanaged, ns, nil
	}

	glog.Warningf("Namespace %q has invalid management label %q", name, value)
	r.recorder.Eventf(
		ns,
		corev1.EventTypeWarning,
		"InvalidManagementLabel",
		"Namespace %q has invalid management label %q",
		name, value,
	)
	*syncErrs = append(*syncErrs, v1.NamespaceConfigSyncError{
		ErrorMessage: invalidManagementLabel(value),
	})
	return namespaceStateUnmanaged, ns, nil
}

func invalidManagementLabel(invalid string) string {
	return fmt.Sprintf("Namespace has invalid management label %s=%s should be in [%s,%s] or unset",
		v1.ResourceManagementKey, invalid,
		v1.ResourceManagementEnabled, v1.ResourceManagementDisabled)
}

func (r *NamespaceConfigReconciler) reconcileNamespaceConfig(
	ctx context.Context,
	name string,
) error {
	var syncErrs []v1.NamespaceConfigSyncError
	cfgState, node := r.getNamespaceConfigState(ctx, name)

	nsState, ns, nsErr := r.getNamespaceState(ctx, name, &syncErrs)
	if nsErr != nil {
		return nsErr
	}

	if glog.V(4) {
		glog.Warningf("ns:%q: cfgState=%v, nsState=%v", name, cfgState, nsState)
	}
	switch cfgState {
	case namespaceConfigStateNotFound:
		if namespaceutil.IsManageable(name) {
			// Special handling for manageable system namespaces: do not remove
			// the namespace itself as that is not allowed.  Instead, manage
			// all policies inside as if the namespace has no managed
			// resources.
			if err := r.managePolicies(
				ctx, name, reservedNamespaceConfig, syncErrs); err != nil {
				return err
			}
			// Remove the metadata from the namespace only after the resources
			// inside have been processed.
			return r.removeNomosMeta(ctx, ns)
		}
		switch nsState {
		case namespaceStateNotFound: // noop
		case namespaceStateUnmanaged, namespaceStateUnset:
			// If unmanaged or managed label unset, do not delete.
			if err := r.cleanUpLabel(ctx, ns); err != nil {
				glog.Warningf("Failed to remove management label from namespace: %s", err.Error())
			}
		case namespaceStateManaged:
			return r.deleteNamespace(ctx, ns)
		}

	case namespaceConfigStateNamespace:
		switch nsState {
		case namespaceStateNotFound:
			if err := r.createNamespace(ctx, node); err != nil {
				syncErrs = append(syncErrs, v1.NamespaceConfigSyncError{
					ErrorMessage: fmt.Sprintf("Failed to create namespace: %s", err.Error()),
				})
				if err2 := r.setNamespaceConfigStatus(ctx, node, syncErrs); err2 != nil {
					glog.Warningf("failed to set status on policy node after ns creation error: %s", err2)
				}
				return err
			}
			return r.managePolicies(ctx, name, node, syncErrs)
		case namespaceStateManaged, namespaceStateUnset:
			if err := r.updateNamespace(ctx, node); err != nil {
				syncErrs = append(syncErrs, v1.NamespaceConfigSyncError{
					ErrorMessage: fmt.Sprintf("Failed to update namespace: %s", err.Error()),
				})
			}
			return r.managePolicies(ctx, name, node, syncErrs)
		case namespaceStateUnmanaged:
			if err := r.cleanUpLabel(ctx, ns); err != nil {
				syncErrs = append(syncErrs, v1.NamespaceConfigSyncError{
					ErrorMessage: fmt.Sprintf("Failed to remove quota label from namespace: %s", err.Error()),
				})
			}
			r.warnUnmanaged(ns)
			syncErrs = append(syncErrs, v1.NamespaceConfigSyncError{ErrorMessage: unmanagedError()})
			return r.managePolicies(ctx, name, node, syncErrs)
		}
	}
	return nil
}

func unmanagedError() string {
	return fmt.Sprintf("Namespace is labeled unmanaged (%s=%s). Must be labeled (%s=%s) to manage",
		v1.ResourceManagementKey, v1.ResourceManagementDisabled, v1.ResourceManagementKey, v1.ResourceManagementEnabled)
}

func (r *NamespaceConfigReconciler) warnUnmanaged(ns *corev1.Namespace) {
	glog.Warningf("namespace %q is declared in the source of truth but is unmanaged in cluster", ns.Name)
	r.recorder.Event(
		ns, corev1.EventTypeWarning, "UnmanagedNamespace",
		"namespace is declared in the source of truth but does is unmanaged")
}

func (r *NamespaceConfigReconciler) warnInvalidAnnotationResource(u *unstructured.Unstructured, msg string) {
	gvk := u.GroupVersionKind()
	value := u.GetAnnotations()[v1.ResourceManagementKey]
	glog.Warningf("%q with name %q is %s in the source of truth but has invalid management annotation %s=%s",
		gvk, u.GetName(), msg, v1.ResourceManagementKey, value)
	r.recorder.Eventf(
		u, corev1.EventTypeWarning, "InvalidAnnotation",
		"%q is %s in the source of truth but has invalid management annotation %s=%s", gvk, v1.ResourceManagementKey, value)
}

// removeNomosMeta removes Nomos-specific decorations from the given namespace.
func (r *NamespaceConfigReconciler) removeNomosMeta(ctx context.Context, ns *corev1.Namespace) error {
	if !v1.HasNomosAnnotation(ns.GetAnnotations()) && !labeling.HasNomos(ns.GetLabels()) {
		// Skip object update if there is nothing to change.
		return nil
	}
	_, err := r.client.Update(ctx, ns, func(o runtime.Object) (runtime.Object, error) {
		ns := o.(*corev1.Namespace)
		// Set GVK for ease of logging.
		ns.SetGroupVersionKind(kinds.Namespace())
		a := ns.GetAnnotations()
		v1.RemoveNomos(a)
		ns.SetAnnotations(a)
		l := ns.GetLabels()
		labeling.RemoveNomos(l)
		ns.SetLabels(l)
		return ns, nil
	})
	return err
}

// cleanUpLabel removes the nomos quota label from the namespace, if present.
func (r *NamespaceConfigReconciler) cleanUpLabel(ctx context.Context, ns *corev1.Namespace) error {
	if _, ok := ns.GetLabels()[labeling.ConfigManagementQuotaKey]; !ok {
		return nil
	}

	_, err := r.client.Update(ctx, ns, func(o runtime.Object) (runtime.Object, error) {
		nso := o.(*corev1.Namespace)
		nso.SetGroupVersionKind(kinds.Namespace())
		ls := nso.GetLabels()
		delete(ls, labeling.ConfigManagementQuotaKey)
		nso.SetLabels(ls)
		return nso, nil
	})
	return err
}

func (r *NamespaceConfigReconciler) managePolicies(ctx context.Context, name string, node *v1.NamespaceConfig, syncErrs []v1.NamespaceConfigSyncError) error {
	var errBuilder status.ErrorBuilder
	reconcileCount := 0
	grs, err := r.decoder.DecodeResources(node.Spec.Resources...)
	if err != nil {
		return errors.Wrapf(err, "could not process namespaceconfig: %q", node.GetName())
	}
	for _, gvk := range r.toSync {
		declaredInstances := grs[gvk]
		decorateAsManaged(declaredInstances, node)
		allDeclaredVersions := allVersionNames(grs, gvk.GroupKind())

		actualInstances, err := r.cache.UnstructuredList(gvk, name)
		if err != nil {
			errBuilder.Add(status.APIServerWrapf(err, "failed to list from policy controller for %q", gvk))
			syncErrs = append(syncErrs, NewSyncError(name, gvk, err))
			continue
		}

		diffs := differ.Diffs(declaredInstances, allDeclaredVersions, actualInstances)
		for _, diff := range diffs {
			if updated, err := r.handleDiff(ctx, diff); err != nil {
				errBuilder.Add(err)
				pse := NewSyncError(name, gvk, err)
				pse.ResourceName = diff.Name
				syncErrs = append(syncErrs, pse)
			} else if updated {
				reconcileCount++
			}
		}
	}
	if err := r.setNamespaceConfigStatus(ctx, node, syncErrs); err != nil {
		errBuilder.Add(errors.Wrapf(err, "failed to set status for %q", name))
		metrics.ErrTotal.WithLabelValues(node.GetName(), node.GroupVersionKind().Kind, "update").Inc()
		r.recorder.Eventf(node, corev1.EventTypeWarning, "StatusUpdateFailed",
			"failed to update policy node status: %s", err)
	}
	if errBuilder.Len() == 0 && reconcileCount > 0 && len(syncErrs) == 0 {
		r.recorder.Eventf(node, corev1.EventTypeNormal, "ReconcileComplete",
			"policy node %q was successfully reconciled: %d changes", name, reconcileCount)
	}
	// TODO(ekitson): Update this function to return MultiError instead of returning explicit nil.
	bErr := errBuilder.Build()
	if bErr == nil {
		return nil
	}
	return bErr
}

// TODO(sbochins): consolidate common functionality with decorateAsClusterManaged.
func decorateAsManaged(declaredInstances []*unstructured.Unstructured, node *v1.NamespaceConfig) {
	for _, decl := range declaredInstances {
		decl.SetNamespace(node.GetName())
		annotateManaged(decl, node.Spec.ImportToken)
	}
}

// setNamespaceConfigStatus updates the status of the given node.  If the node
// is nil, it does nothing, and successfully so.
func (r *NamespaceConfigReconciler) setNamespaceConfigStatus(
	ctx context.Context,
	node *v1.NamespaceConfig,
	errs []v1.NamespaceConfigSyncError) id.ResourceError {
	if node == reservedNamespaceConfig {
		return nil
	}
	freshSyncToken := node.Status.SyncToken == node.Spec.ImportToken
	if node.Status.SyncState.IsSynced() && freshSyncToken && len(errs) == 0 {
		glog.Infof("Status for NamespaceConfig %q is already up-to-date.", node.Name)
		return nil
	}

	updateFn := func(obj runtime.Object) (runtime.Object, error) {
		newPN := obj.(*v1.NamespaceConfig)
		newPN.Status.SyncToken = node.Spec.ImportToken
		newPN.Status.SyncTime = now()
		newPN.Status.SyncErrors = errs
		if len(errs) > 0 {
			newPN.Status.SyncState = v1.StateError
		} else {
			newPN.Status.SyncState = v1.StateSynced
		}
		newPN.SetGroupVersionKind(kinds.NamespaceConfig())
		return newPN, nil
	}
	_, err := r.client.UpdateStatus(ctx, node, updateFn)
	return err
}

// NewSyncError returns a NamespaceConfigSyncError corresponding to the given error and action
func NewSyncError(name string, gvk schema.GroupVersionKind, err error) v1.NamespaceConfigSyncError {
	return v1.NamespaceConfigSyncError{
		SourceName:   name,
		ResourceKind: gvk.Kind,
		ResourceAPI:  gvk.GroupVersion().String(),
		ErrorMessage: err.Error(),
	}
}

// handleDiff updates the API Server according to changes reflected in the diff.
// It returns whether or not an update occurred and the error encountered.
func (r *NamespaceConfigReconciler) handleDiff(ctx context.Context, diff *differ.Diff) (bool, id.ResourceError) {
	switch diff.Type {
	case differ.Add:
		return r.handleAdd(ctx, diff)
	case differ.Update:
		switch diff.ManagementState() {
		case differ.Managed, differ.Unset:
			// Manage if managed or management label is unset.
			return r.handleUpdate(diff)
		case differ.Unmanaged:
			return false, nil
		case differ.Invalid:
			r.warnInvalidAnnotationResource(diff.Actual, "declared")
			return false, nil
		}
	case differ.Delete:
		switch diff.ManagementState() {
		case differ.Managed:
			return r.handleDelete(ctx, diff)
		case differ.Unmanaged, differ.Unset:
			// Don't delete if unmanaged or management label is unset.
			return false, nil
		case differ.Invalid:
			r.warnInvalidAnnotationResource(diff.Actual, "not declared")
			return false, nil
		}
	default:
		panic(fmt.Errorf("programmatic error, unhandled syncer diff type: %v", diff.Type))
	}
	return true, nil
}

func (r *NamespaceConfigReconciler) handleAdd(ctx context.Context, diff *differ.Diff) (bool, id.ResourceError) {
	toCreate := diff.Declared
	if err := r.applier.Create(ctx, toCreate); err != nil {
		metrics.ErrTotal.WithLabelValues(toCreate.GetNamespace(), toCreate.GetKind(), "create").Inc()
		return false, id.ResourceWrap(err, fmt.Sprintf("failed to create %q", diff.Name), ast.ParseFileObject(toCreate))
	}
	return true, nil
}

func (r *NamespaceConfigReconciler) handleUpdate(diff *differ.Diff) (bool, id.ResourceError) {
	ns := diff.Declared.GetNamespace()
	if err := r.applier.ApplyNamespace(ns, diff.Declared, diff.Actual); err != nil {
		metrics.ErrTotal.WithLabelValues(ns, diff.Declared.GetKind(), "patch").Inc()
		return false, id.ResourceWrap(err, fmt.Sprintf("failed to patch %q", diff.Name), ast.ParseFileObject(diff.Declared))
	}
	return true, nil
}

func (r *NamespaceConfigReconciler) handleDelete(ctx context.Context, diff *differ.Diff) (bool, id.ResourceError) {
	toDelete := diff.Actual
	if err := r.client.Delete(ctx, toDelete); err != nil {
		metrics.ErrTotal.WithLabelValues(toDelete.GetNamespace(), toDelete.GetKind(), "delete").Inc()
		return false, id.ResourceWrap(err, fmt.Sprintf("failed to delete %q", diff.Name), ast.ParseFileObject(toDelete))
	}
	return true, nil
}

func asNamespace(namespaceConfig *v1.NamespaceConfig) *corev1.Namespace {
	return withNamespaceConfigMeta(&corev1.Namespace{}, namespaceConfig)
}

func withNamespaceConfigMeta(namespace *corev1.Namespace, namespaceConfig *v1.NamespaceConfig) *corev1.Namespace {
	namespace.SetGroupVersionKind(kinds.Namespace())
	if !namespaceutil.IsSystem(namespace.GetName()) {
		// Mark the namespace as supporting the management of hierarchical quota.
		// But don't interfere with system namespaces, since that could lock us
		// out of the cluster.
		labels := labeling.ManageQuota.AddDeepCopy(namespaceConfig.Labels)
		namespace.Labels = labels
	}

	if namespace.Annotations == nil {
		namespace.Annotations = make(map[string]string)
	}
	for k, v := range namespaceConfig.Annotations {
		namespace.Annotations[k] = v
	}

	namespace.Annotations[v1.ResourceManagementKey] = v1.ResourceManagementEnabled
	namespace.Name = namespaceConfig.Name
	namespace.SetGroupVersionKind(kinds.Namespace())
	return namespace
}

func (r *NamespaceConfigReconciler) createNamespace(ctx context.Context, namespaceConfig *v1.NamespaceConfig) error {
	namespace := asNamespace(namespaceConfig)
	if err := r.client.Create(ctx, namespace); err != nil {
		metrics.ErrTotal.WithLabelValues(namespace.GetName(), namespace.GroupVersionKind().Kind, "create").Inc()
		r.recorder.Eventf(namespaceConfig, corev1.EventTypeWarning, "NamespaceCreateFailed",
			"failed to create namespace: %q", err)
		return errors.Wrapf(err, "failed to create namespace %q", namespaceConfig.Name)
	}
	return nil
}

func (r *NamespaceConfigReconciler) updateNamespace(ctx context.Context, namespaceConfig *v1.NamespaceConfig) error {
	glog.V(1).Infof("Namespace %q declared in a policy node, updating", namespaceConfig.Name)

	namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespaceConfig.Name}}
	namespace.SetGroupVersionKind(kinds.Namespace())
	updateFn := func(obj runtime.Object) (runtime.Object, error) {
		return withNamespaceConfigMeta(obj.(*corev1.Namespace), namespaceConfig), nil
	}
	if _, err := r.client.Update(ctx, namespace, updateFn); err != nil {
		metrics.ErrTotal.WithLabelValues(namespace.GetName(), namespace.GroupVersionKind().Kind, "update").Inc()
		r.recorder.Eventf(namespaceConfig, corev1.EventTypeWarning, "NamespaceUpdateFailed",
			"failed to update namespace: %q", err)
		return errors.Wrapf(err, "failed to update namespace %q", namespaceConfig.Name)
	}
	return nil
}

func (r *NamespaceConfigReconciler) deleteNamespace(ctx context.Context, namespace *corev1.Namespace) error {
	glog.V(1).Infof("Namespace %q not declared in a policy node, removing", namespace.GetName())
	if err := r.client.Delete(ctx, namespace); err != nil {
		metrics.ErrTotal.WithLabelValues(namespace.GetName(), namespace.GroupVersionKind().Kind, "delete").Inc()
		return errors.Wrapf(err, "failed to delete namespace %q", namespace.GetName())
	}
	return nil
}
