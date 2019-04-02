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
	"github.com/google/nomos/pkg/api/configmanagement/v1"
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

// getNamespaceConfig normalizes the state of the policy node and returns the node.
func (r *NamespaceConfigReconciler) getNamespaceConfig(
	ctx context.Context,
	name string,
) *v1.NamespaceConfig {
	node := &v1.NamespaceConfig{}
	err := r.cache.Get(ctx, apitypes.NamespacedName{Name: name}, node)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		panic(errors.Wrap(err, "cache returned error other than not found, this should not happen"))
	}
	node.SetGroupVersionKind(kinds.NamespaceConfig())

	return node
}

// getNamespace normalizes the state of the namespace and retrieves the current value.
func (r *NamespaceConfigReconciler) getNamespace(
	ctx context.Context,
	name string,
	syncErrs *[]v1.ConfigManagementError,
) (*corev1.Namespace, error) {
	ns := &corev1.Namespace{}
	err := r.cache.Get(ctx, apitypes.NamespacedName{Name: name}, ns)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "got unhandled lister error")
	}

	return ns, nil
}

func invalidManagementLabel(invalid string) string {
	return fmt.Sprintf("Namespace has invalid management annotation %s=%s should be %q or unset",
		v1.ResourceManagementKey, invalid, v1.ResourceManagementEnabled)
}

func (r *NamespaceConfigReconciler) reconcileNamespaceConfig(
	ctx context.Context,
	name string,
) error {
	var syncErrs []v1.ConfigManagementError
	node := r.getNamespaceConfig(ctx, name)

	ns, nsErr := r.getNamespace(ctx, name, &syncErrs)
	if nsErr != nil {
		return nsErr
	}

	diff := differ.NamespaceDiff{
		Name:     name,
		Declared: node,
		Actual:   ns,
	}

	if glog.V(4) {
		glog.Warningf("ns:%q: diffType=%v", name, diff.Type())
	}
	switch diff.Type() {
	case differ.Create:
		if err := r.createNamespace(ctx, node); err != nil {
			syncErrs = append(syncErrs, NewSyncError(node, err))

			if err2 := r.setNamespaceConfigStatus(ctx, node, syncErrs); err2 != nil {
				glog.Warningf("failed to set status on policy node after ns creation error: %s", err2)
			}
			return err
		}
		return r.managePolicies(ctx, name, node, syncErrs)

	case differ.Update:
		if err := r.updateNamespace(ctx, node); err != nil {
			syncErrs = append(syncErrs, NewSyncError(node, err))
		}
		return r.managePolicies(ctx, name, node, syncErrs)

	case differ.Delete:
		if namespaceutil.IsManageableSystem(name) {
			// Special handling for manageable system namespaces: do not remove
			// the namespace itself as that is not allowed.  Instead, manage
			// all policies inside as if the namespace has no managed
			// resources.
			if err := r.managePolicies(ctx, name, reservedNamespaceConfig, syncErrs); err != nil {
				return err
			}
			// Remove the metadata from the namespace only after the resources
			// inside have been processed.
			return r.unmanageNamespace(ctx, ns)
		}
		return r.deleteNamespace(ctx, ns)

	case differ.Unmanage:
		// Remove defunct labels and annotations.
		unmanageErr := r.unmanageNamespace(ctx, ns)
		if unmanageErr != nil {
			glog.Warningf("Failed to remove quota label and management annotations from namespace: %s", unmanageErr.Error())
			return unmanageErr
		}
		if node != nil {
			r.warnUnmanaged(ns)
			syncErrs = append(syncErrs, cmeForNamespace(ns, unmanagedError()))
		}

		// Return an error if any encountered.
		if reconcileErr := r.managePolicies(ctx, name, node, syncErrs); reconcileErr != nil {
			return reconcileErr
		}
		return unmanageErr

	case differ.Error:
		value := node.GetAnnotations()[v1.ResourceManagementKey]
		glog.Warningf("Namespace %q has invalid management annotation %q", name, value)
		r.recorder.Eventf(
			node,
			corev1.EventTypeWarning,
			"InvalidManagementLabel",
			"Namespace %q has invalid management annotation %q",
			name, value,
		)
		syncErrs = append(syncErrs, cmeForNamespace(ns, invalidManagementLabel(value)))
		return r.managePolicies(ctx, name, node, syncErrs)

	case differ.NoOp:
		if node != nil {
			r.warnUnmanaged(ns)
			syncErrs = append(syncErrs, cmeForNamespace(ns, unmanagedError()))
		}
		return r.managePolicies(ctx, name, node, syncErrs)
	}
	panic(fmt.Sprintf("unhandled diff type: %v", diff.Type()))
}

func unmanagedError() string {
	return fmt.Sprintf("Namespace annotated unmanaged (%s=%s) in repo. Must not have %s annotation to manage",
		v1.ResourceManagementKey, v1.ResourceManagementDisabled, v1.ResourceManagementKey)
}

func (r *NamespaceConfigReconciler) warnUnmanaged(ns *corev1.Namespace) {
	glog.Warningf("namespace %q is declared in the source of truth but is unmanaged in cluster", ns.Name)
	r.recorder.Event(
		ns, corev1.EventTypeWarning, "UnmanagedNamespace",
		"namespace is declared in the source of truth but does is unmanaged")
}

// unmanageNamespace removes the nomos annotations and labels from the Namespace.
func (r *NamespaceConfigReconciler) unmanageNamespace(ctx context.Context, ns *corev1.Namespace) error {
	_, err := r.client.Update(ctx, ns, func(o runtime.Object) (runtime.Object, error) {
		nso := o.(*corev1.Namespace)
		nso.GetObjectKind().SetGroupVersionKind(kinds.Namespace())
		removeNomosMeta(nso)
		return nso, nil
	})
	return err
}

func (r *NamespaceConfigReconciler) managePolicies(ctx context.Context, name string, node *v1.NamespaceConfig, syncErrs []v1.ConfigManagementError) error {
	if node == nil {
		return nil
	}
	var errBuilder status.ErrorBuilder
	reconcileCount := 0
	grs, err := r.decoder.DecodeResources(node.Spec.Resources...)
	if err != nil {
		return errors.Wrapf(err, "could not process namespaceconfig: %q", node.GetName())
	}
	for _, gvk := range r.toSync {
		declaredInstances := grs[gvk]
		for _, decl := range declaredInstances {
			decl.SetNamespace(node.GetName())
			annotate(decl, kv(v1.SyncTokenAnnotationKey, node.Spec.Token))
		}

		actualInstances, err := r.cache.UnstructuredList(gvk, name)
		if err != nil {
			errBuilder.Add(status.APIServerWrapf(err, "failed to list from policy controller for %q", gvk))
			syncErrs = append(syncErrs, NewSyncError(node, err))
			continue
		}

		allDeclaredVersions := allVersionNames(grs, gvk.GroupKind())
		diffs := differ.Diffs(declaredInstances, actualInstances, allDeclaredVersions)
		for _, diff := range diffs {
			if updated, err := handleDiff(ctx, r.applier, diff, r.recorder); err != nil {
				errBuilder.Add(err)
				syncErrs = append(syncErrs, cmesForResourceError(err)...)
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

// setNamespaceConfigStatus updates the status of the given node.  If the node
// is nil, it does nothing, and successfully so.
func (r *NamespaceConfigReconciler) setNamespaceConfigStatus(
	ctx context.Context,
	node *v1.NamespaceConfig,
	errs []v1.ConfigManagementError) id.ResourceError {
	if node == reservedNamespaceConfig {
		return nil
	}
	freshSyncToken := node.Status.Token == node.Spec.Token
	if node.Status.SyncState.IsSynced() && freshSyncToken && len(errs) == 0 {
		glog.Infof("Status for NamespaceConfig %q is already up-to-date.", node.Name)
		return nil
	}

	updateFn := func(obj runtime.Object) (runtime.Object, error) {
		newPN := obj.(*v1.NamespaceConfig)
		newPN.Status.Token = node.Spec.Token
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

// NewSyncError returns a ConfigManagementError corresponding to the given NamespaceConfig and error
func NewSyncError(config *v1.NamespaceConfig, err error) v1.ConfigManagementError {
	return v1.ConfigManagementError{
		SourcePath:        config.GetAnnotations()[v1.SourcePathAnnotationKey],
		ResourceName:      config.GetName(),
		ResourceNamespace: config.GetNamespace(),
		ResourceGVK:       config.GroupVersionKind(),
		ErrorMessage:      err.Error(),
	}
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

	for k, v := range namespaceConfig.Annotations {
		annotate(namespace, kv(k, v))
	}
	annotate(namespace, kv(v1.ResourceManagementKey, v1.ResourceManagementEnabled))

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
