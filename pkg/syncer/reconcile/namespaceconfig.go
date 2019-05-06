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
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/cache"
	"github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/decode"
	"github.com/google/nomos/pkg/syncer/differ"
	"github.com/google/nomos/pkg/syncer/metrics"
	"github.com/google/nomos/pkg/util/namespaceutil"
	"github.com/pkg/errors"
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

	// reservedNamespaceConfig is a dummy namespace config used to represent the config content of
	// non-removable namespaces (like "default") when the corresponding NamespaceConfig is deleted
	// from the repo. Instead of deleting the namespace and its resources, we apply a change that
	// removes all managed resources from the namespace, but does not attempt to delete the namespace.
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
	now      func() metav1.Time
	toSync   []schema.GroupVersionKind
	// A cancelable ambient context for all reconciler operations.
	ctx context.Context
}

// NewNamespaceConfigReconciler returns a new NamespaceConfigReconciler.
func NewNamespaceConfigReconciler(ctx context.Context, client *client.Client, applier Applier, cache cache.GenericCache, recorder record.EventRecorder,
	decoder decode.Decoder, now func() metav1.Time, toSync []schema.GroupVersionKind) *NamespaceConfigReconciler {
	return &NamespaceConfigReconciler{
		client:   client,
		applier:  applier,
		cache:    cache,
		recorder: recorder,
		decoder:  decoder,
		toSync:   toSync,
		now:      now,
		ctx:      ctx,
	}
}

// Reconcile is the Reconcile callback for NamespaceConfigReconciler.
func (r *NamespaceConfigReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	start := r.now()
	metrics.ReconcileEventTimes.WithLabelValues("namespace").Set(float64(start.Unix()))

	name := request.Name
	glog.Infof("Reconciling NamespaceConfig: %q", name)
	if namespaceutil.IsReserved(name) {
		glog.Errorf("Trying to reconcile a NamespaceConfig corresponding to a reserved namespace: %q", name)
		// We don't return an error, because we should never be reconciling these NamespaceConfigs in the first place.
		return reconcile.Result{}, nil
	}

	ctx, cancel := context.WithTimeout(r.ctx, reconcileTimeout)
	defer cancel()

	err := r.reconcileNamespaceConfig(ctx, name)
	metrics.ReconcileDuration.WithLabelValues("namespace", metrics.StatusLabel(err)).Observe(time.Since(start.Time).Seconds())

	if err != nil {
		glog.Errorf("Could not reconcile namespaceconfig %q: %v", name, err)
	}
	return reconcile.Result{}, err
}

// getNamespaceConfig normalizes the state of the NamespaceConfig and returns the config.
func (r *NamespaceConfigReconciler) getNamespaceConfig(
	ctx context.Context,
	name string,
) *v1.NamespaceConfig {
	config := &v1.NamespaceConfig{}
	err := r.cache.Get(ctx, apitypes.NamespacedName{Name: name}, config)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		panic(errors.Wrap(err, "cache returned error other than not found, this should not happen"))
	}
	config.SetGroupVersionKind(kinds.NamespaceConfig())

	return config
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
	config := r.getNamespaceConfig(ctx, name)

	ns, nsErr := r.getNamespace(ctx, name, &syncErrs)
	if nsErr != nil {
		return nsErr
	}

	diff := differ.NamespaceDiff{
		Name:     name,
		Declared: config,
		Actual:   ns,
	}

	if glog.V(4) {
		glog.Warningf("ns:%q: diffType=%v", name, diff.Type())
	}
	switch diff.Type() {
	case differ.Create:
		if err := r.createNamespace(ctx, config); err != nil {
			syncErrs = append(syncErrs, NewSyncError(config, err))

			if err2 := r.setNamespaceConfigStatus(ctx, config, syncErrs); err2 != nil {
				glog.Warningf("Failed to set status on NamespaceConfig after namespace creation error: %s", err2)
			}
			return err
		}
		return r.manageConfigs(ctx, name, config, syncErrs)

	case differ.Update:
		if err := r.updateNamespace(ctx, config); err != nil {
			syncErrs = append(syncErrs, NewSyncError(config, err))
		}
		return r.manageConfigs(ctx, name, config, syncErrs)

	case differ.Delete:
		if namespaceutil.IsManageableSystem(name) {
			// Special handling for manageable system namespaces: do not remove the namespace itself as
			// that is not allowed.  Instead, manage all configs inside as if the namespace has no managed
			// resources.
			if err := r.manageConfigs(ctx, name, reservedNamespaceConfig, syncErrs); err != nil {
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
		if config != nil {
			r.warnUnmanaged(ns)
			syncErrs = append(syncErrs, cmeForNamespace(ns, unmanagedError()))
		}

		// Return an error if any encountered.
		if reconcileErr := r.manageConfigs(ctx, name, config, syncErrs); reconcileErr != nil {
			return reconcileErr
		}
		return unmanageErr

	case differ.Error:
		value := config.GetAnnotations()[v1.ResourceManagementKey]
		glog.Warningf("Namespace %q has invalid management annotation %q", name, value)
		r.recorder.Eventf(
			config,
			corev1.EventTypeWarning,
			"InvalidManagementLabel",
			"Namespace %q has invalid management annotation %q",
			name, value,
		)
		syncErrs = append(syncErrs, cmeForNamespace(ns, invalidManagementLabel(value)))
		return r.manageConfigs(ctx, name, config, syncErrs)

	case differ.NoOp:
		if config != nil {
			r.warnUnmanaged(ns)
			syncErrs = append(syncErrs, cmeForNamespace(ns, unmanagedError()))
		}
		return r.manageConfigs(ctx, name, config, syncErrs)
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

func (r *NamespaceConfigReconciler) manageConfigs(ctx context.Context, name string, config *v1.NamespaceConfig, syncErrs []v1.ConfigManagementError) error {
	if config == nil {
		return nil
	}
	var errBuilder status.MultiError
	reconcileCount := 0
	grs, err := r.decoder.DecodeResources(config.Spec.Resources...)
	if err != nil {
		return errors.Wrapf(err, "could not process namespaceconfig: %q", config.GetName())
	}
	for _, gvk := range r.toSync {
		declaredInstances := grs[gvk]
		for _, decl := range declaredInstances {
			decl.SetNamespace(config.GetName())
			object.SetAnnotation(decl, v1.SyncTokenAnnotationKey, config.Spec.Token)
		}

		actualInstances, err := r.cache.UnstructuredList(gvk, name)
		if err != nil {
			errBuilder = status.Append(errBuilder, status.APIServerWrapf(err, "failed to list from NamespaceConfig controller for %q", gvk))
			syncErrs = append(syncErrs, NewSyncError(config, err))
			continue
		}

		allDeclaredVersions := AllVersionNames(grs, gvk.GroupKind())
		diffs := differ.Diffs(declaredInstances, actualInstances, allDeclaredVersions)
		for _, diff := range diffs {
			if updated, err := HandleDiff(ctx, r.applier, diff, r.recorder); err != nil {
				errBuilder = status.Append(errBuilder, err)
				syncErrs = append(syncErrs, err.ToCME())
			} else if updated {
				reconcileCount++
			}
		}
	}
	if err := r.setNamespaceConfigStatus(ctx, config, syncErrs); err != nil {
		errBuilder = status.Append(errBuilder, errors.Wrapf(err, "failed to set status for NamespaceConfig %q", name))
		r.recorder.Eventf(config, corev1.EventTypeWarning, "StatusUpdateFailed",
			"failed to update NamespaceConfig status: %s", err)
	}
	if errBuilder == nil && reconcileCount > 0 && len(syncErrs) == 0 {
		r.recorder.Eventf(config, corev1.EventTypeNormal, "ReconcileComplete",
			"NamespaceConfig %q was successfully reconciled: %d changes", name, reconcileCount)
	}
	return errBuilder
}

// setNamespaceConfigStatus updates the status of the given NamespaceConfig. If the config is nil,
// it does nothing, and successfully so.
func (r *NamespaceConfigReconciler) setNamespaceConfigStatus(
	ctx context.Context, config *v1.NamespaceConfig, errs []v1.ConfigManagementError) status.Error {
	if config == reservedNamespaceConfig {
		return nil
	}
	freshSyncToken := config.Status.Token == config.Spec.Token
	if config.Status.SyncState.IsSynced() && freshSyncToken && len(errs) == 0 {
		glog.Infof("Status for NamespaceConfig %q is already up-to-date.", config.Name)
		return nil
	}

	updateFn := func(obj runtime.Object) (runtime.Object, error) {
		newPN := obj.(*v1.NamespaceConfig)
		newPN.Status.Token = config.Spec.Token
		newPN.Status.SyncTime = r.now()
		newPN.Status.SyncErrors = errs
		if len(errs) > 0 {
			newPN.Status.SyncState = v1.StateError
		} else {
			newPN.Status.SyncState = v1.StateSynced
		}
		newPN.SetGroupVersionKind(kinds.NamespaceConfig())
		return newPN, nil
	}
	_, err := r.client.UpdateStatus(ctx, config, updateFn)
	// TODO(fmil): Missing error monitoring like util.go/SetClusterConfigStatus.
	return err
}

// NewSyncError returns a ConfigManagementError corresponding to the given NamespaceConfig and error
func NewSyncError(config *v1.NamespaceConfig, err error) v1.ConfigManagementError {
	e := v1.ErrorResource{
		SourcePath:        config.GetAnnotations()[v1.SourcePathAnnotationKey],
		ResourceName:      config.GetName(),
		ResourceNamespace: config.GetNamespace(),
		ResourceGVK:       config.GroupVersionKind(),
	}
	cme := v1.ConfigManagementError{
		ErrorMessage: err.Error(),
	}
	cme.ErrorResources = append(cme.ErrorResources, e)
	return cme
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
		namespace.SetLabels(nil)
		for k, v := range namespaceConfig.Labels {
			object.SetLabel(namespace, k, v)
		}
		enableQuota(namespace)
	}

	namespace.SetAnnotations(nil)
	for k, v := range namespaceConfig.Annotations {
		object.SetAnnotation(namespace, k, v)
	}
	EnableManagement(namespace)
	object.SetAnnotation(namespace, v1.SyncTokenAnnotationKey, namespaceConfig.Spec.Token)

	namespace.Name = namespaceConfig.Name
	namespace.SetGroupVersionKind(kinds.Namespace())
	return namespace
}

func (r *NamespaceConfigReconciler) createNamespace(ctx context.Context, namespaceConfig *v1.NamespaceConfig) error {
	namespace := asNamespace(namespaceConfig)
	err := r.client.Create(ctx, namespace)
	metrics.Operations.WithLabelValues("create", namespace.Kind, metrics.StatusLabel(err)).Inc()

	if err != nil {
		r.recorder.Eventf(namespaceConfig, corev1.EventTypeWarning, "NamespaceCreateFailed",
			"failed to create namespace: %q", err)
		return errors.Wrapf(err, "failed to create namespace %q", namespaceConfig.Name)
	}
	return nil
}

func (r *NamespaceConfigReconciler) updateNamespace(ctx context.Context, namespaceConfig *v1.NamespaceConfig) error {
	glog.V(1).Infof("Namespace %q declared in a NamespaceConfig, updating", namespaceConfig.Name)

	namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespaceConfig.Name}}
	namespace.SetGroupVersionKind(kinds.Namespace())
	updateFn := func(obj runtime.Object) (runtime.Object, error) {
		return withNamespaceConfigMeta(obj.(*corev1.Namespace), namespaceConfig), nil
	}

	_, err := r.client.Update(ctx, namespace, updateFn)
	metrics.Operations.WithLabelValues("update", namespace.Kind, metrics.StatusLabel(err)).Inc()

	if err != nil {
		r.recorder.Eventf(namespaceConfig, corev1.EventTypeWarning, "NamespaceUpdateFailed",
			"failed to update namespace: %q", err)
		return errors.Wrapf(err, "failed to update namespace %q", namespaceConfig.Name)
	}
	return nil
}

func (r *NamespaceConfigReconciler) deleteNamespace(ctx context.Context, namespace *corev1.Namespace) error {
	glog.V(1).Infof("Namespace %q not declared in a NamespaceConfig, removing", namespace.GetName())

	err := r.client.Delete(ctx, namespace)
	metrics.Operations.WithLabelValues("create", namespace.Kind, metrics.StatusLabel(err)).Inc()

	if err != nil {
		return errors.Wrapf(err, "failed to delete namespace %q", namespace.GetName())
	}
	return nil
}
