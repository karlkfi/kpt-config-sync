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
	"time"

	"github.com/golang/glog"
	nomosv1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/generic-syncer/cache"
	"github.com/google/nomos/pkg/generic-syncer/client"
	"github.com/google/nomos/pkg/generic-syncer/decode"
	"github.com/google/nomos/pkg/generic-syncer/differ"
	"github.com/google/nomos/pkg/generic-syncer/labeling"
	"github.com/google/nomos/pkg/generic-syncer/metrics"
	"github.com/google/nomos/pkg/util/multierror"
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

var _ reconcile.Reconciler = &PolicyNodeReconciler{}

// PolicyNodeReconciler reconciles a PolicyNode object.
type PolicyNodeReconciler struct {
	client     *client.Client
	cache      cache.GenericCache
	recorder   record.EventRecorder
	decoder    decode.Decoder
	comparator *differ.Comparator
	toSync     []schema.GroupVersionKind
}

// NewPolicyNodeReconciler returns a new PolicyNodeReconciler.
func NewPolicyNodeReconciler(client *client.Client, cache cache.GenericCache, recorder record.EventRecorder,
	decoder decode.Decoder, comparator *differ.Comparator, toSync []schema.GroupVersionKind) *PolicyNodeReconciler {
	return &PolicyNodeReconciler{
		client:     client,
		cache:      cache,
		recorder:   recorder,
		decoder:    decoder,
		comparator: comparator,
		toSync:     toSync,
	}
}

// Reconcile is the Reconcile callback for PolicyNodeReconciler.
func (r *PolicyNodeReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	metrics.EventTimes.WithLabelValues("reconcilePolicyNode").Set(float64(now().Unix()))
	reconcileTimer := prometheus.NewTimer(
		metrics.NamespaceReconcileDuration.WithLabelValues(request.Name))
	defer reconcileTimer.ObserveDuration()

	ctx, cancel := context.WithTimeout(context.Background(), reconcileTimeout)
	defer cancel()
	// TODO(sbochins): Make use of reconcile.Result.RequeueAfter when we don't want exponential backoff for retries when
	// using newer version of controller-runtime.
	glog.Infof("Reconciling Policy Node: %s", request.Name)
	err := r.reconcilePolicyNode(ctx, request.Name)
	if err != nil {
		glog.Errorf("Could not reconcile policynode %q: %v", request.Name, err)
	}
	return reconcile.Result{}, err
}

// policyNodeState enumerates possible states for PolicyNodes
type policyNodeState string

const (
	policyNodeStateNotFound    = policyNodeState("notFound")    // the policy node does not exist
	policyNodeStateNamespace   = policyNodeState("namespace")   // the policy node is declared as a namespace
	policyNodeStatePolicyspace = policyNodeState("policyspace") // the policy node is declared as a policyspace
	policyNodeStateReserved    = policyNodeState("reserved")    // the policy node is declared as a reserved namespace
)

// getPolicyNodeState normalizes the state of the policy node and returns the node.
func (r *PolicyNodeReconciler) getPolicyNodeState(ctx context.Context, name string) (policyNodeState, *nomosv1.PolicyNode,
	error) {
	node := &nomosv1.PolicyNode{}
	err := r.cache.Get(ctx, apitypes.NamespacedName{Name: name}, node)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return policyNodeStateNotFound, nil, nil
		}
		panic(errors.Wrap(err, "cache returned error other than not found, this should not happen"))
	}

	if namespaceutil.IsReserved(name) {
		return policyNodeStateReserved, node, nil
	}

	switch node.Spec.Type {
	case nomosv1.Policyspace:
		return policyNodeStatePolicyspace, node, nil
	case nomosv1.Namespace:
		return policyNodeStateNamespace, node, nil
	case nomosv1.ReservedNamespace:
		return policyNodeStateReserved, node, nil
	default:
		return policyNodeStateNotFound, nil, errors.Errorf("invalid node type %q", node.Spec.Type)
	}
}

// namespaceState enumerates possible states for the namespace
type namespaceState string

const (
	namespaceStateNotFound       = namespaceState("notFound")       // the namespace does not exist
	namespaceStateExists         = namespaceState("exists")         // the namespace exists
	namespaceStateManagePolicies = namespaceState("managePolicies") // the namespace is labeled for policy management
	namespaceStateManageFull     = namespaceState("manageFull")     // the namespace is labeled for policy and lifecycle management
)

// getNamespaceState normalizes the state of the namespace and retrieves the current value.
func (r *PolicyNodeReconciler) getNamespaceState(ctx context.Context, name string) (namespaceState, *corev1.Namespace,
	error) {
	ns := &corev1.Namespace{}
	err := r.cache.Get(ctx, apitypes.NamespacedName{Name: name}, ns)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return namespaceStateNotFound, nil, nil
		}
		return namespaceStateNotFound, nil, errors.Wrapf(err, "got unhandled lister error")
	}

	value, found := ns.Labels[labeling.ManagementKey]
	if !found {
		return namespaceStateExists, ns, nil
	}

	switch value {
	case labeling.Policies:
		return namespaceStateManagePolicies, ns, nil
	case labeling.Full:
		return namespaceStateManageFull, ns, nil
	}

	glog.Warningf("Namespace %q has invalid management label %q", name, value)
	r.recorder.Eventf(
		ns,
		corev1.EventTypeWarning,
		"InvalidManagementLabel",
		"Namespace %q has invalid management label %q",
		name, value,
	)
	return namespaceStateExists, ns, nil
}

func (r *PolicyNodeReconciler) reconcilePolicyNode(ctx context.Context, name string) error {
	pnState, node, pnErr := r.getPolicyNodeState(ctx, name)
	if pnErr != nil {
		return pnErr
	}

	nsState, ns, nsErr := r.getNamespaceState(ctx, name)
	if nsErr != nil {
		return nsErr
	}

	switch pnState {
	case policyNodeStateNotFound:
		if namespaceutil.IsReserved(name) {
			return nil
		}
		switch nsState {
		case namespaceStateNotFound: // noop
		case namespaceStateExists:
			r.warnUndeclaredNamespace(ns)
		case namespaceStateManagePolicies:
			r.warnUndeclaredNamespace(ns)
		case namespaceStateManageFull:
			return r.deleteNamespace(ctx, ns)
		}

	case policyNodeStateNamespace:
		switch nsState {
		case namespaceStateNotFound:
			if err := r.createNamespace(ctx, node); err != nil {
				return err
			}
			return r.managePolicies(ctx, name, node)
		case namespaceStateExists:
			r.warnNoLabel(ns)
			syncErrs := []nomosv1.PolicyNodeSyncError{
				{
					ErrorMessage: fmt.Sprintf("Namespace is missing proper management label (%s={%s,%s})",
						labeling.ManagementKey, labeling.Policies, labeling.Full),
				},
			}
			if err := r.setPolicyNodeStatus(ctx, node, syncErrs); err != nil {
				return err
			}
		case namespaceStateManagePolicies:
			if err := r.updateNamespaceLabels(ctx, node); err != nil {
				return err
			}
			return r.managePolicies(ctx, name, node)
		case namespaceStateManageFull:
			if err := r.updateNamespace(ctx, node); err != nil {
				return err
			}
			return r.managePolicies(ctx, name, node)
		}

	case policyNodeStatePolicyspace:
		switch nsState {
		case namespaceStateNotFound: // noop
		case namespaceStateExists:
			r.warnPolicyspaceHasNamespace(ns)
		case namespaceStateManagePolicies:
			r.warnPolicyspaceHasNamespace(ns)
		case namespaceStateManageFull:
			return r.handlePolicyspace(ctx, node)
		}

	case policyNodeStateReserved:
		switch nsState {
		case namespaceStateNotFound: // noop
		case namespaceStateExists: // noop
		case namespaceStateManagePolicies:
			r.warnReservedLabel(ns)
		case namespaceStateManageFull:
			r.warnReservedLabel(ns)
		}
	}
	return nil
}

func (r *PolicyNodeReconciler) warnUndeclaredNamespace(ns *corev1.Namespace) {
	glog.Warningf("namespace %q exists but is not declared in the source of truth", ns.Name)
	r.recorder.Event(
		ns, corev1.EventTypeWarning, "UnmanagedNamespace",
		"namespace is not declared in the source of truth")
}

func (r *PolicyNodeReconciler) warnPolicyspaceHasNamespace(ns *corev1.Namespace) {
	glog.Warningf("namespace %q exists but is declared as a policyspace in the source of truth", ns.Name)
	r.recorder.Event(
		ns, corev1.EventTypeWarning, "NamespaceInPolicySpace",
		"namespace is declared as a policyspace in the source of truth")
}

func (r *PolicyNodeReconciler) warnReservedLabel(ns *corev1.Namespace) {
	glog.Warningf("reserved namespace %q has a management label", ns.Name)
	r.recorder.Event(
		ns, corev1.EventTypeWarning, "UnmanagedNamespace",
		"reserved namespace has a management label")
}

func (r *PolicyNodeReconciler) warnNoLabel(ns *corev1.Namespace) {
	glog.Warningf("namespace %q is declared in the source of truth but does not have a management label", ns.Name)
	r.recorder.Event(
		ns, corev1.EventTypeWarning, "UnmanagedNamespace",
		"namespace is declared in the source of truth but does not have a management label")
}

func (r *PolicyNodeReconciler) warnNoLabelResource(u *unstructured.Unstructured) {
	gvk := u.GroupVersionKind()
	glog.Warningf("%q with name %q is declared in the source of truth but does not have a management label",
		gvk, u.GetName())
	r.recorder.Eventf(
		u, corev1.EventTypeWarning, "UnmanagedResource",
		"%q is declared in the source of truth but does not have a management label", gvk)
}

func (r *PolicyNodeReconciler) managePolicies(ctx context.Context, name string, node *nomosv1.PolicyNode) error {
	var syncErrs []nomosv1.PolicyNodeSyncError
	var errBuilder multierror.Builder
	reconcileCount := 0
	grs, err := r.decoder.DecodeResources(node.Spec.Resources...)
	if err != nil {
		return errors.Wrapf(err, "could not process policynode: %q", node.GetName())
	}
	for _, gvk := range r.toSync {
		declaredInstances := grs[gvk]
		for _, decl := range declaredInstances {
			decl.SetNamespace(name)
			// Label the resource as Nomos managed.
			decl.SetLabels(labeling.ManageResource.AddDeepCopy(decl.GetLabels()))
			// Annotate the resource with the current version token.
			a := decl.GetAnnotations()
			if a == nil {
				a = map[string]string{v1alpha1.SyncTokenAnnotationKey: node.Spec.ImportToken}
			} else {
				a[v1alpha1.SyncTokenAnnotationKey] = node.Spec.ImportToken
			}
			decl.SetAnnotations(a)
		}

		actualInstances, err := r.cache.UnstructuredList(gvk, name)
		if err != nil {
			errBuilder.Add(errors.Wrapf(err, "failed to list from policy controller for %q", gvk))
			syncErrs = append(syncErrs, NewSyncError(name, gvk, err))
			continue
		}

		diffs := differ.Diffs(r.comparator.Equal, declaredInstances, actualInstances)
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
	if err := r.setPolicyNodeStatus(ctx, node, syncErrs); err != nil {
		errBuilder.Add(errors.Wrapf(err, "failed to set status for %q", name))
		metrics.ErrTotal.WithLabelValues(node.GetName(), node.GetObjectKind().GroupVersionKind().Kind, "update").Inc()
		r.recorder.Eventf(node, corev1.EventTypeWarning, "StatusUpdateFailed",
			"failed to update policy node status: %q", err)
	}
	if errBuilder.Len() == 0 && reconcileCount > 0 {
		r.recorder.Eventf(node, corev1.EventTypeNormal, "ReconcileComplete",
			"policy node was successfully reconciled: %d changes", reconcileCount)
	}
	return errBuilder.Build()
}

func (r *PolicyNodeReconciler) setPolicyNodeStatus(ctx context.Context, node *nomosv1.PolicyNode,
	errs []nomosv1.PolicyNodeSyncError) error {
	freshSyncToken := node.Status.SyncToken == node.Spec.ImportToken
	if node.Status.SyncState.IsSynced() && freshSyncToken && len(errs) == 0 {
		glog.Infof("Status for PolicyNode %q is already up-to-date.", node.Name)
		return nil
	}

	updateFn := func(obj runtime.Object) (runtime.Object, error) {
		newPN := obj.(*nomosv1.PolicyNode)
		newPN.Status.SyncToken = node.Spec.ImportToken
		newPN.Status.SyncTime = now()
		newPN.Status.SyncErrors = errs
		if len(errs) > 0 {
			newPN.Status.SyncState = nomosv1.StateError
		} else {
			newPN.Status.SyncState = nomosv1.StateSynced
		}
		return newPN, nil
	}
	// TODO(ekitson): Use UpdateStatus() when our minimum supported k8s version is 1.11.
	_, err := r.client.Update(ctx, node, updateFn)
	return err
}

// NewSyncError returns a PolicyNodeSyncError corresponding to the given error and action
func NewSyncError(name string, gvk schema.GroupVersionKind, err error) nomosv1.PolicyNodeSyncError {
	return nomosv1.PolicyNodeSyncError{
		SourceName:   name,
		ResourceKind: gvk.Kind,
		ResourceAPI:  gvk.GroupVersion().String(),
		ErrorMessage: err.Error(),
	}
}

// handleDiff updates the API Server according to changes reflected in the diff.
// It returns whether or not an update occurred and the error encountered.
func (r *PolicyNodeReconciler) handleDiff(ctx context.Context, diff *differ.Diff) (bool, error) {
	switch t := diff.Type; t {
	case differ.Add:
		toCreate := diff.Declared
		if err := r.client.Create(ctx, toCreate); err != nil {
			metrics.ErrTotal.WithLabelValues(toCreate.GetNamespace(), toCreate.GetKind(), "create").Inc()
			return false, errors.Wrapf(err, "could not create resource %q", diff.Name)
		}
	case differ.Update:
		if !diff.ActualResourceIsManaged() {
			r.warnNoLabelResource(diff.Actual)
			return false, nil
		}

		toUpdate := diff.Declared
		toUpdate.SetResourceVersion(diff.Actual.GetResourceVersion())
		if err := r.client.Upsert(ctx, toUpdate); err != nil {
			metrics.ErrTotal.WithLabelValues(toUpdate.GetNamespace(), toUpdate.GetKind(), "update").Inc()
			return false, errors.Wrapf(err, "could not update resource %q", diff.Name)
		}
	case differ.Delete:
		if !diff.ActualResourceIsManaged() {
			r.warnNoLabelResource(diff.Actual)
			return false, nil
		}
		toDelete := diff.Actual
		if err := r.client.Delete(ctx, toDelete); err != nil {
			metrics.ErrTotal.WithLabelValues(toDelete.GetNamespace(), toDelete.GetKind(), "delete").Inc()
			return false, errors.Wrapf(err, "could not delete resource %q", diff.Name)
		}
	default:
		panic(fmt.Errorf("programmatic error, unhandled syncer diff type: %v", t))
	}
	return true, nil
}

func asNamespace(policyNode *nomosv1.PolicyNode) *corev1.Namespace {
	return withPolicyNodeMeta(&corev1.Namespace{}, policyNode)
}

func withPolicyNodeMeta(namespace *corev1.Namespace, policyNode *nomosv1.PolicyNode) *corev1.Namespace {
	labels := labeling.ManageAll.AddDeepCopy(policyNode.Labels)
	namespace.Labels = labels
	namespace.Annotations = policyNode.Annotations
	namespace.Name = policyNode.Name
	namespace.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Namespace"))
	return namespace
}

func (r *PolicyNodeReconciler) createNamespace(ctx context.Context, policyNode *nomosv1.PolicyNode) error {
	namespace := asNamespace(policyNode)
	if err := r.client.Create(ctx, namespace); err != nil {
		metrics.ErrTotal.WithLabelValues(namespace.GetName(), namespace.GetObjectKind().GroupVersionKind().Kind, "create").Inc()
		r.recorder.Eventf(policyNode, corev1.EventTypeWarning, "NamespaceCreateFailed",
			"failed to create matching namespace for policyspace: %q", err)
		return errors.Wrapf(err, "failed to create namespace %q", policyNode.Name)
	}
	return nil
}

func (r *PolicyNodeReconciler) updateNamespace(ctx context.Context, policyNode *nomosv1.PolicyNode) error {
	glog.V(1).Infof("Namespace %q declared in a policy node, updating", policyNode.Name)

	namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: policyNode.Name}}
	updateFn := func(obj runtime.Object) (runtime.Object, error) {
		return withPolicyNodeMeta(obj.(*corev1.Namespace), policyNode), nil
	}
	if _, err := r.client.Update(ctx, namespace, updateFn); err != nil {
		metrics.ErrTotal.WithLabelValues(namespace.GetName(), namespace.GetObjectKind().GroupVersionKind().Kind, "update").Inc()
		r.recorder.Eventf(policyNode, corev1.EventTypeWarning, "NamespaceUpdateFailed",
			"failed to update matching namespace for policyspace: %q", err)
		return errors.Wrapf(err, "failed to update namespace %q", policyNode.Name)
	}
	return nil
}

// updateNamespaceLabels is used for updating the parent label on a namespace where we manage policy values
// This is used since we can't update all the labels on the namespace.
func (r *PolicyNodeReconciler) updateNamespaceLabels(ctx context.Context, policyNode *nomosv1.PolicyNode) error {
	labels := map[string]string{}

	namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: policyNode.Name}}
	updateFn := func(obj runtime.Object) (runtime.Object, error) {
		ns := obj.(*corev1.Namespace)
		for key, value := range labels {
			if oldValue, found := ns.Labels[key]; !found || oldValue != value {
				ns.Labels[key] = value
			}
		}
		return ns, nil
	}
	if _, err := r.client.Update(ctx, namespace, updateFn); err != nil {
		metrics.ErrTotal.WithLabelValues(namespace.GetName(), namespace.GetObjectKind().GroupVersionKind().Kind, "update").Inc()
		r.recorder.Eventf(policyNode, corev1.EventTypeWarning, "NamespaceUpdateFailed",
			"failed to update matching namespace for policyspace: %v", err)
		return errors.Wrapf(err, "failed to execute update action for %q", namespace.Name)
	}

	return nil
}

func (r *PolicyNodeReconciler) deleteNamespace(ctx context.Context, namespace *corev1.Namespace) error {
	glog.V(1).Infof("Namespace %q not declared in a policy node, removing", namespace.GetName())
	if err := r.client.Delete(ctx, namespace); err != nil {
		metrics.ErrTotal.WithLabelValues(namespace.GetName(), namespace.GetObjectKind().GroupVersionKind().Kind, "delete").Inc()
		return errors.Wrapf(err, "failed to delete namespace %q", namespace.GetName())
	}
	return nil
}

func (r *PolicyNodeReconciler) handlePolicyspace(ctx context.Context, policyNode *nomosv1.PolicyNode) error {
	namespace := asNamespace(policyNode)
	if err := r.client.Delete(ctx, namespace); err != nil {
		metrics.ErrTotal.WithLabelValues("", namespace.GetObjectKind().GroupVersionKind().Kind, "delete").Inc()
		r.recorder.Eventf(policyNode, corev1.EventTypeWarning, "NamespaceDeleteFailed",
			"failed to delete matching namespace for policyspace: %v", err)
		return errors.Wrapf(err, "failed to delete policyspace %q", policyNode.Name)
	}
	r.recorder.Event(policyNode, corev1.EventTypeNormal, "NamespaceDeleted",
		"removed matching namespace for policyspace")
	return nil
}
