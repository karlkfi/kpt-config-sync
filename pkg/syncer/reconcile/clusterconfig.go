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

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/id"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/cache"
	"github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/decode"
	"github.com/google/nomos/pkg/syncer/differ"
	"github.com/google/nomos/pkg/syncer/metrics"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ reconcile.Reconciler = &ClusterConfigReconciler{}

// now is stubbed out in unit tests.
var now = metav1.Now

// ClusterConfigReconciler reconciles a ClusterConfig object.
type ClusterConfigReconciler struct {
	client   *client.Client
	applier  Applier
	cache    cache.GenericCache
	recorder record.EventRecorder
	decoder  decode.Decoder
	toSync   []schema.GroupVersionKind
	// A cancelable ambient context for all reconciler operations.
	ctx context.Context
}

// NewClusterConfigReconciler returns a new ClusterConfigReconciler.  ctx is the ambient context
// to use for all reconciler operations.
func NewClusterConfigReconciler(ctx context.Context, client *client.Client, applier Applier, cache cache.GenericCache, recorder record.EventRecorder,
	decoder decode.Decoder, toSync []schema.GroupVersionKind) *ClusterConfigReconciler {
	return &ClusterConfigReconciler{
		client:   client,
		applier:  applier,
		cache:    cache,
		recorder: recorder,
		decoder:  decoder,
		toSync:   toSync,
		ctx:      ctx,
	}
}

// Reconcile is the Reconcile callback for ClusterConfigReconciler.
func (r *ClusterConfigReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	metrics.EventTimes.WithLabelValues("cluster-reconcile").Set(float64(now().Unix()))
	timer := prometheus.NewTimer(metrics.ClusterReconcileDuration.WithLabelValues())
	defer timer.ObserveDuration()

	ctx, cancel := context.WithTimeout(r.ctx, reconcileTimeout)
	defer cancel()

	clusterConfig := &v1.ClusterConfig{}
	err := r.cache.Get(ctx, request.NamespacedName, clusterConfig)
	if err != nil {
		err = errors.Wrapf(err, "could not retrieve clusterconfig %q", request.Name)
		glog.Error(err)
		return reconcile.Result{}, err
	}
	clusterConfig.SetGroupVersionKind(kinds.ClusterConfig())

	name := request.Name
	if request.Name != v1.ClusterConfigName {
		r.recorder.Eventf(clusterConfig, corev1.EventTypeWarning, "InvalidClusterConfig",
			"ClusterConfig resource has invalid name %q", name)
		err := errors.Errorf("ClusterConfig resource has invalid name %q", name)
		glog.Warning(err)
		// Only return an error if we cannot update the status,
		// since we don't want kubebuilder to enqueue a retry for this object.
		return reconcile.Result{}, r.setClusterConfigStatus(ctx, clusterConfig, NewClusterConfigSyncError(name, clusterConfig.GroupVersionKind(), err))
	}

	rErr := r.managePolicies(ctx, clusterConfig)
	if rErr != nil {
		glog.Errorf("Could not reconcile clusterconfig: %v", rErr)
	}
	return reconcile.Result{}, rErr
}

func (r *ClusterConfigReconciler) managePolicies(ctx context.Context, policy *v1.ClusterConfig) error {
	grs, err := r.decoder.DecodeResources(policy.Spec.Resources...)
	if err != nil {
		return errors.Wrapf(err, "could not process cluster policy: %q", policy.GetName())
	}

	name := policy.GetName()
	var syncErrs []v1.ClusterConfigSyncError
	var errBuilder status.ErrorBuilder
	reconcileCount := 0
	for _, gvk := range r.toSync {
		declaredInstances := grs[gvk]
		for _, decl := range declaredInstances {
			annotateManaged(decl, policy.Spec.ImportToken)
		}
		allDeclaredVersions := allVersionNames(grs, gvk.GroupKind())

		actualInstances, err := r.cache.UnstructuredList(gvk, "")
		if err != nil {
			errBuilder.Add(status.APIServerWrapf(err, "failed to list from policy controller for %q", gvk))
			syncErrs = append(syncErrs, NewClusterConfigSyncError(name, gvk, err))
			continue
		}

		diffs := differ.Diffs(declaredInstances, allDeclaredVersions, actualInstances)
		for _, diff := range diffs {
			if updated, err := r.handleDiff(ctx, diff); err != nil {
				errBuilder.Add(err)
				pse := NewClusterConfigSyncError(name, gvk, err)
				pse.ResourceName = diff.Name
				syncErrs = append(syncErrs, pse)
			} else if updated {
				reconcileCount++
			}
		}
	}
	if err := r.setClusterConfigStatus(ctx, policy, syncErrs...); err != nil {
		errBuilder.Add(err)
		r.recorder.Eventf(policy, corev1.EventTypeWarning, "StatusUpdateFailed",
			"failed to update cluster policy status: %v", err)
	}
	if errBuilder.Len() == 0 && reconcileCount > 0 {
		r.recorder.Eventf(policy, corev1.EventTypeNormal, "ReconcileComplete",
			"cluster policy was successfully reconciled: %d changes", reconcileCount)
	}
	// TODO(ekitson): Update this function to return MultiError instead of returning explicit nil.
	bErr := errBuilder.Build()
	if bErr == nil {
		return nil
	}
	return bErr
}

func (r *ClusterConfigReconciler) setClusterConfigStatus(ctx context.Context, policy *v1.ClusterConfig,
	errs ...v1.ClusterConfigSyncError) id.ResourceError {
	freshSyncToken := policy.Status.SyncToken == policy.Spec.ImportToken
	if policy.Status.SyncState.IsSynced() && freshSyncToken && len(errs) == 0 {
		glog.Infof("Status for ClusterConfig %q is already up-to-date.", policy.Name)
		return nil
	}

	updateFn := func(obj runtime.Object) (runtime.Object, error) {
		newPolicy := obj.(*v1.ClusterConfig)
		newPolicy.Status.SyncToken = policy.Spec.ImportToken
		newPolicy.Status.SyncTime = now()
		newPolicy.Status.SyncErrors = errs
		if len(errs) > 0 {
			newPolicy.Status.SyncState = v1.StateError
		} else {
			newPolicy.Status.SyncState = v1.StateSynced
		}
		return newPolicy, nil
	}
	_, err := r.client.UpdateStatus(ctx, policy, updateFn)
	if err != nil {
		metrics.ErrTotal.WithLabelValues("", policy.GroupVersionKind().Kind, "update").Inc()
	}
	return err
}

// NewClusterConfigSyncError returns a ClusterConfigSyncError corresponding to the given error and GroupVersionKind.
func NewClusterConfigSyncError(name string, gvk schema.GroupVersionKind, err error) v1.ClusterConfigSyncError {
	return v1.ClusterConfigSyncError{
		ResourceName: name,
		ResourceKind: gvk.Kind,
		ResourceAPI:  gvk.GroupVersion().String(),
		ErrorMessage: err.Error(),
	}
}

// handleDiff updates the API Server according to changes reflected in the diff.
// It returns whether or not an update occurred and the error encountered.
func (r *ClusterConfigReconciler) handleDiff(ctx context.Context, diff *differ.Diff) (bool, id.ResourceError) {
	switch diff.Type {
	case differ.Add:
		return r.handleAdd(ctx, diff)

	case differ.Update:
		switch diff.ManagementState() {
		case differ.Managed, differ.Unset:
			// Update the resource if
			// - it management annotation is "managed", or
			// - it exists in the repository and has no managed annotation.
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
			// Do nothing if managed annotation is unset or explicitly marked "disabled".
			return false, nil
		case differ.Invalid:
			r.warnInvalidAnnotationResource(diff.Actual, "not declared")
			return false, nil
		}
	}
	panic(vet.InternalErrorf("programmatic error, unhandled syncer diff type combination: %v and %v", diff.Type, diff.ManagementState()))
}

func (r *ClusterConfigReconciler) handleAdd(ctx context.Context, diff *differ.Diff) (bool, id.ResourceError) {
	toCreate := diff.Declared
	if err := r.applier.Create(ctx, toCreate); err != nil {
		metrics.ErrTotal.WithLabelValues(toCreate.GetNamespace(), toCreate.GetKind(), "create").Inc()
		return false, id.ResourceWrap(err, fmt.Sprintf("failed to create %q", diff.Name), ast.ParseFileObject(toCreate))
	}
	return true, nil
}

func (r *ClusterConfigReconciler) handleUpdate(diff *differ.Diff) (bool, id.ResourceError) {
	removeEmptyRulesField(diff.Declared)
	if err := r.applier.ApplyCluster(diff.Declared, diff.Actual); err != nil {
		metrics.ErrTotal.WithLabelValues("", diff.Declared.GroupVersionKind().Kind, "patch").Inc()
		return false, id.ResourceWrap(err, fmt.Sprintf("failed to patch %q", diff.Name), ast.ParseFileObject(diff.Declared))
	}
	return true, nil
}

func (r *ClusterConfigReconciler) handleDelete(ctx context.Context, diff *differ.Diff) (bool, id.ResourceError) {
	toDelete := diff.Actual
	if err := r.client.Delete(ctx, toDelete); err != nil {
		metrics.ErrTotal.WithLabelValues("", toDelete.GroupVersionKind().Kind, "delete").Inc()
		return false, id.ResourceWrap(err, fmt.Sprintf("failed to delete %q", diff.Name), ast.ParseFileObject(toDelete))
	}
	return true, nil
}

func (r *ClusterConfigReconciler) warnInvalidAnnotationResource(u *unstructured.Unstructured, msg string) {
	gvk := u.GroupVersionKind()
	value := u.GetAnnotations()[v1.ResourceManagementKey]
	glog.Warningf("%q with name %q is %s in the source of truth but has invalid management annotation %s=%s",
		gvk, u.GetName(), msg, v1.ResourceManagementKey, value)
	r.recorder.Eventf(
		u, corev1.EventTypeWarning, "InvalidAnnotation",
		"%q is %s in the source of truth but has invalid management annotation %s=%s", gvk, v1.ResourceManagementKey, value)
}

// removeEmptyRulesField removes the Rules field from ClusterRole when it's an empty list.
// This is to ensure that we don't overwrite PolicyRules generated by other controllers
// for aggregated ClusterRoles when we `apply` changes.
func removeEmptyRulesField(u *unstructured.Unstructured) {
	if u.GroupVersionKind() != kinds.ClusterRole() {
		return
	}

	if rules, ok := u.Object["rules"]; ok && (rules == nil || len(rules.([]interface{})) == 0) {
		delete(u.Object, "rules")
	}
}
