/*
Copyright 2018 The Nomos Authors.
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
	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/syncer/cache"
	"github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/decode"
	"github.com/google/nomos/pkg/syncer/differ"
	"github.com/google/nomos/pkg/syncer/metrics"
	"github.com/google/nomos/pkg/util/multierror"
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

var _ reconcile.Reconciler = &ClusterPolicyReconciler{}

// now is stubbed out in unit tests.
var now = metav1.Now

// ClusterPolicyReconciler reconciles a ClusterPolicy object.
type ClusterPolicyReconciler struct {
	client   *client.Client
	applier  Applier
	cache    cache.GenericCache
	recorder record.EventRecorder
	decoder  decode.Decoder
	toSync   []schema.GroupVersionKind
}

// NewClusterPolicyReconciler returns a new ClusterPolicyReconciler.
func NewClusterPolicyReconciler(client *client.Client, applier Applier, cache cache.GenericCache, recorder record.EventRecorder,
	decoder decode.Decoder, toSync []schema.GroupVersionKind) *ClusterPolicyReconciler {
	return &ClusterPolicyReconciler{
		client:   client,
		applier:  applier,
		cache:    cache,
		recorder: recorder,
		decoder:  decoder,
		toSync:   toSync,
	}
}

// Reconcile is the Reconcile callback for ClusterPolicyReconciler.
func (r *ClusterPolicyReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	metrics.EventTimes.WithLabelValues("cluster-reconcile").Set(float64(now().Unix()))
	timer := prometheus.NewTimer(metrics.ClusterReconcileDuration.WithLabelValues())
	defer timer.ObserveDuration()

	ctx, cancel := context.WithTimeout(context.Background(), reconcileTimeout)
	defer cancel()

	clusterPolicy := &v1.ClusterPolicy{}
	err := r.cache.Get(ctx, request.NamespacedName, clusterPolicy)
	if err != nil {
		err = errors.Wrapf(err, "could not retrieve clusterpolicy %q", request.Name)
		glog.Error(err)
		return reconcile.Result{}, err
	}

	name := request.Name
	if request.Name != v1.ClusterPolicyName {
		r.recorder.Eventf(clusterPolicy, corev1.EventTypeWarning, "InvalidClusterPolicy",
			"ClusterPolicy resource has invalid name %q", name)
		err := errors.Errorf("ClusterPolicy resource has invalid name %q", name)
		glog.Warning(err)
		// Only return an error if we cannot update the status,
		// since we don't want kubebuilder to enqueue a retry for this object.
		return reconcile.Result{}, r.setClusterPolicyStatus(ctx, clusterPolicy, NewClusterPolicySyncError(name, clusterPolicy.GroupVersionKind(), err))
	}

	rErr := r.managePolicies(ctx, clusterPolicy)
	if rErr != nil {
		glog.Errorf("Could not reconcile clusterpolicy: %v", rErr)
	}
	return reconcile.Result{}, rErr
}

func (r *ClusterPolicyReconciler) managePolicies(ctx context.Context, policy *v1.ClusterPolicy) error {
	grs, err := r.decoder.DecodeResources(policy.Spec.Resources...)
	if err != nil {
		return errors.Wrapf(err, "could not process cluster policy: %q", policy.GetName())
	}

	name := policy.GetName()
	var syncErrs []v1.ClusterPolicySyncError
	var errBuilder multierror.Builder
	reconcileCount := 0
	for _, gvk := range r.toSync {
		declaredInstances := grs[gvk]
		decorateAsClusterManaged(declaredInstances, policy)
		allDeclaredVersions := allVersionNames(grs, gvk.GroupKind())

		actualInstances, err := r.cache.UnstructuredList(gvk, "")
		if err != nil {
			errBuilder.Add(errors.Wrapf(err, "failed to list from policy controller for %q", gvk))
			syncErrs = append(syncErrs, NewClusterPolicySyncError(name, gvk, err))
			continue
		}

		diffs := differ.Diffs(declaredInstances, allDeclaredVersions, actualInstances)
		for _, diff := range diffs {
			if updated, err := r.handleDiff(ctx, diff); err != nil {
				errBuilder.Add(err)
				pse := NewClusterPolicySyncError(name, gvk, err)
				pse.ResourceName = diff.Name
				syncErrs = append(syncErrs, pse)
			} else if updated {
				reconcileCount++
			}
		}
	}
	if err := r.setClusterPolicyStatus(ctx, policy, syncErrs...); err != nil {
		errBuilder.Add(errors.Wrapf(err, "failed to set status for %q", name))
		r.recorder.Eventf(policy, corev1.EventTypeWarning, "StatusUpdateFailed",
			"failed to update cluster policy status: %q", err)
	}
	if errBuilder.Len() == 0 && reconcileCount > 0 {
		r.recorder.Eventf(policy, corev1.EventTypeNormal, "ReconcileComplete",
			"cluster policy was successfully reconciled: %d changes", reconcileCount)
	}
	return errBuilder.Build()
}

func decorateAsClusterManaged(declaredInstances []*unstructured.Unstructured, policy *v1.ClusterPolicy) {
	for _, decl := range declaredInstances {
		// Annotate the resource with the current version token.
		a := decl.GetAnnotations()
		if a == nil {
			a = map[string]string{}
		}
		// Annotate the resource with the current version token.
		a[v1alpha1.SyncTokenAnnotationKey] = policy.Spec.ImportToken
		// Annotate the resource as Nomos managed.
		a[v1alpha1.ResourceManagementKey] = v1alpha1.ResourceManagementValue
		decl.SetAnnotations(a)
	}
}

func (r *ClusterPolicyReconciler) setClusterPolicyStatus(ctx context.Context, policy *v1.ClusterPolicy,
	errs ...v1.ClusterPolicySyncError) error {
	freshSyncToken := policy.Status.SyncToken == policy.Spec.ImportToken
	if policy.Status.SyncState.IsSynced() && freshSyncToken && len(errs) == 0 {
		glog.Infof("Status for ClusterPolicy %q is already up-to-date.", policy.Name)
		return nil
	}

	updateFn := func(obj runtime.Object) (runtime.Object, error) {
		newPolicy := obj.(*v1.ClusterPolicy)
		newPolicy.Status.SyncToken = policy.Spec.ImportToken
		newPolicy.Status.SyncTime = now()
		newPolicy.Status.SyncErrors = errs
		if len(errs) > 0 {
			newPolicy.Status.SyncState = v1.StateError
		} else {
			newPolicy.Status.SyncState = v1.StateSynced
		}
		newPolicy.SetGroupVersionKind(kinds.ClusterPolicy())
		return newPolicy, nil
	}
	_, err := r.client.UpdateStatus(ctx, policy, updateFn)
	if err != nil {
		metrics.ErrTotal.WithLabelValues("", policy.GroupVersionKind().Kind, "update").Inc()
	}
	return err
}

// NewClusterPolicySyncError returns a ClusterPolicySyncError corresponding to the given error and GroupVersionKind.
func NewClusterPolicySyncError(name string, gvk schema.GroupVersionKind, err error) v1.ClusterPolicySyncError {
	return v1.ClusterPolicySyncError{
		ResourceName: name,
		ResourceKind: gvk.Kind,
		ResourceAPI:  gvk.GroupVersion().String(),
		ErrorMessage: err.Error(),
	}
}

// handleDiff updates the API Server according to changes reflected in the diff.
// It returns whether or not an update occurred and the error encountered.
func (r *ClusterPolicyReconciler) handleDiff(ctx context.Context, diff *differ.Diff) (bool, error) {
	switch t := diff.Type; t {
	case differ.Add:
		toCreate := diff.Declared
		if err := r.applier.Create(ctx, toCreate); err != nil {
			metrics.ErrTotal.WithLabelValues(toCreate.GetNamespace(), toCreate.GetKind(), "create").Inc()
			return false, errors.Wrapf(err, "could not create resource %q", diff.Name)
		}
	case differ.Update:
		switch diff.ActualResourceIsManaged() {
		case differ.Managed:
		case differ.Unmanaged:
			return false, nil
		case differ.Invalid:
			r.warnInvalidAnnotationResource(diff.Actual, "declared")
			return false, nil
		}

		removeEmptyRulesField(diff.Declared)
		if err := r.applier.ApplyCluster(diff.Declared, diff.Actual); err != nil {
			metrics.ErrTotal.WithLabelValues("", diff.Declared.GroupVersionKind().Kind, "patch").Inc()
			return false, err
		}
	case differ.Delete:
		switch diff.ActualResourceIsManaged() {
		case differ.Managed:
		case differ.Unmanaged:
			return false, nil
		case differ.Invalid:
			r.warnInvalidAnnotationResource(diff.Actual, "not declared")
			return false, nil
		}

		toDelete := diff.Actual
		if err := r.client.Delete(ctx, toDelete); err != nil {
			metrics.ErrTotal.WithLabelValues("", toDelete.GroupVersionKind().Kind, "delete").Inc()
			return false, errors.Wrapf(err, "could not delete resource %q", diff.Name)
		}
	default:
		panic(fmt.Errorf("programmatic error, unhandled syncer diff type: %v", t))
	}
	return true, nil
}

func (r *ClusterPolicyReconciler) warnInvalidAnnotationResource(u *unstructured.Unstructured, msg string) {
	gvk := u.GroupVersionKind()
	value := u.GetAnnotations()[v1alpha1.ResourceManagementKey]
	glog.Warningf("%q with name %q is %s in the source of truth but has invalid management annotation %s=%s",
		gvk, u.GetName(), msg, v1alpha1.ResourceManagementKey, value)
	r.recorder.Eventf(
		u, corev1.EventTypeWarning, "InvalidAnnotation",
		"%q is %s in the source of truth but has invalid management annotation %s=%s", gvk, v1alpha1.ResourceManagementKey, value)
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
