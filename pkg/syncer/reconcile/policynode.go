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
	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/syncer/cache"
	"github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/decode"
	"github.com/google/nomos/pkg/syncer/differ"
	"github.com/google/nomos/pkg/syncer/labeling"
	"github.com/google/nomos/pkg/syncer/metrics"
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
	client   *client.Client
	applier  Applier
	cache    cache.GenericCache
	recorder record.EventRecorder
	decoder  decode.Decoder
	toSync   []schema.GroupVersionKind
}

// NewPolicyNodeReconciler returns a new PolicyNodeReconciler.
func NewPolicyNodeReconciler(client *client.Client, applier Applier, cache cache.GenericCache, recorder record.EventRecorder,
	decoder decode.Decoder, toSync []schema.GroupVersionKind) *PolicyNodeReconciler {
	return &PolicyNodeReconciler{
		client:   client,
		applier:  applier,
		cache:    cache,
		recorder: recorder,
		decoder:  decoder,
		toSync:   toSync,
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

	name := request.Name
	glog.Infof("Reconciling Policy Node: %q", name)
	if namespaceutil.IsReserved(name) {
		glog.Errorf("Trying to reconcile a PolicyNode corresponding to a reserved namespace: %q", name)
		// We don't return an error, because we should never be reconciling these PolicyNodes in the first place.
		return reconcile.Result{}, nil
	}

	err := r.reconcilePolicyNode(ctx, name)
	if err != nil {
		glog.Errorf("Could not reconcile policynode %q: %v", name, err)
	}
	return reconcile.Result{}, err
}

// policyNodeState enumerates possible states for PolicyNodes
type policyNodeState string

const (
	policyNodeStateNotFound    = policyNodeState("notFound")    // the policy node does not exist
	policyNodeStateNamespace   = policyNodeState("namespace")   // the policy node is declared as a namespace
	policyNodeStatePolicyspace = policyNodeState("policyspace") // the policy node is declared as a policyspace
)

// getPolicyNodeState normalizes the state of the policy node and returns the node.
func (r *PolicyNodeReconciler) getPolicyNodeState(ctx context.Context, name string) (policyNodeState, *v1.PolicyNode,
	error) {
	node := &v1.PolicyNode{}
	err := r.cache.Get(ctx, apitypes.NamespacedName{Name: name}, node)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return policyNodeStateNotFound, nil, nil
		}
		panic(errors.Wrap(err, "cache returned error other than not found, this should not happen"))
	}
	node.SetGroupVersionKind(kinds.PolicyNode())

	switch node.Spec.Type {
	case v1.Policyspace:
		return policyNodeStatePolicyspace, node, nil
	case v1.Namespace:
		return policyNodeStateNamespace, node, nil
	default:
		return policyNodeStateNotFound, nil, errors.Errorf("invalid node type %q", node.Spec.Type)
	}
}

// namespaceState enumerates possible states for the namespace
type namespaceState string

const (
	namespaceStateNotFound   = namespaceState("notFound")   // the namespace does not exist
	namespaceStateExists     = namespaceState("exists")     // the namespace exists and we should manage policies
	namespaceStateManageFull = namespaceState("manageFull") // the namespace is labeled for policy and lifecycle management
)

// getNamespaceState normalizes the state of the namespace and retrieves the current value.
func (r *PolicyNodeReconciler) getNamespaceState(
	ctx context.Context,
	name string,
	syncErrs *[]v1.PolicyNodeSyncError) (namespaceState, *corev1.Namespace,
	error) {
	ns := &corev1.Namespace{}
	err := r.cache.Get(ctx, apitypes.NamespacedName{Name: name}, ns)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return namespaceStateNotFound, nil, nil
		}
		return namespaceStateNotFound, nil, errors.Wrapf(err, "got unhandled lister error")
	}

	value, found := ns.Annotations[v1.ResourceManagementKey]
	if !found {
		return namespaceStateExists, ns, nil
	}

	if value == v1.ResourceManagementValue {
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
	*syncErrs = append(*syncErrs, v1.PolicyNodeSyncError{
		ErrorMessage: fmt.Sprintf("Namespace has invalid management label %s=%s should be %s=%s or unset",
			v1.ResourceManagementKey, value,
			v1.ResourceManagementKey, v1.ResourceManagementValue),
	})
	return namespaceStateExists, ns, nil
}

func (r *PolicyNodeReconciler) reconcilePolicyNode(
	ctx context.Context,
	name string) error {
	var syncErrs []v1.PolicyNodeSyncError
	pnState, node, pnErr := r.getPolicyNodeState(ctx, name)
	if pnErr != nil {
		return pnErr
	}

	nsState, ns, nsErr := r.getNamespaceState(ctx, name, &syncErrs)
	if nsErr != nil {
		return nsErr
	}

	switch pnState {
	case policyNodeStateNotFound:
		switch nsState {
		case namespaceStateNotFound: // noop
		case namespaceStateExists:
			if err := r.cleanUpLabel(ctx, ns); err != nil {
				glog.Warningf("Failed to remove management label from namespace: %s", err.Error())
			}
		case namespaceStateManageFull:
			return r.deleteNamespace(ctx, ns)
		}

	case policyNodeStateNamespace:
		switch nsState {
		case namespaceStateNotFound:
			if err := r.createNamespace(ctx, node); err != nil {
				syncErrs = append(syncErrs, v1.PolicyNodeSyncError{
					ErrorMessage: fmt.Sprintf("Failed to create namespace: %s", err.Error()),
				})
				if err2 := r.setPolicyNodeStatus(ctx, node, syncErrs); err2 != nil {
					glog.Warningf("failed to set status on policy node after ns creation error: %s", err2)
				}
				return err
			}
			return r.managePolicies(ctx, name, node, syncErrs)
		case namespaceStateExists:
			if err := r.cleanUpLabel(ctx, ns); err != nil {
				syncErrs = append(syncErrs, v1.PolicyNodeSyncError{
					ErrorMessage: fmt.Sprintf("Failed to remove quota label from namespace: %s", err.Error()),
				})
			}
			r.warnNoAnnotation(ns)
			syncErrs = append(syncErrs, v1.PolicyNodeSyncError{
				ErrorMessage: fmt.Sprintf("Namespace is missing proper management annotation (%s=%s)",
					v1.ResourceManagementKey, v1.ResourceManagementValue),
			})
			return r.managePolicies(ctx, name, node, syncErrs)
		case namespaceStateManageFull:
			if err := r.updateNamespace(ctx, node); err != nil {
				syncErrs = append(syncErrs, v1.PolicyNodeSyncError{
					ErrorMessage: fmt.Sprintf("Failed to update namespace: %s", err.Error()),
				})
			}
			return r.managePolicies(ctx, name, node, syncErrs)
		}

	case policyNodeStatePolicyspace:
		switch nsState {
		case namespaceStateNotFound: // noop
		case namespaceStateExists:
		case namespaceStateManageFull:
			return r.handlePolicyspace(ctx, node)
		}
	}
	return nil
}

func (r *PolicyNodeReconciler) warnNoAnnotation(ns *corev1.Namespace) {
	glog.Warningf("namespace %q is declared in the source of truth but does not have a management annotation", ns.Name)
	r.recorder.Event(
		ns, corev1.EventTypeWarning, "UnmanagedNamespace",
		"namespace is declared in the source of truth but does not have a management annotation")
}

func (r *PolicyNodeReconciler) warnInvalidAnnotationResource(u *unstructured.Unstructured, msg string) {
	gvk := u.GroupVersionKind()
	value := u.GetAnnotations()[v1.ResourceManagementKey]
	glog.Warningf("%q with name %q is %s in the source of truth but has invalid management annotation %s=%s",
		gvk, u.GetName(), msg, v1.ResourceManagementKey, value)
	r.recorder.Eventf(
		u, corev1.EventTypeWarning, "InvalidAnnotation",
		"%q is %s in the source of truth but has invalid management annotation %s=%s", gvk, v1.ResourceManagementKey, value)
}

// cleanUpLabel removes the nomos quota label from the namespace, if present.
func (r *PolicyNodeReconciler) cleanUpLabel(ctx context.Context, ns *corev1.Namespace) error {
	if _, ok := ns.GetLabels()[labeling.NomosQuotaKey]; !ok {
		return nil
	}

	_, err := r.client.Update(ctx, ns, func(o runtime.Object) (runtime.Object, error) {
		nso := o.(*corev1.Namespace)
		nso.SetGroupVersionKind(kinds.Namespace())
		ls := nso.GetLabels()
		delete(ls, labeling.NomosQuotaKey)
		nso.SetLabels(ls)
		return nso, nil
	})
	return err
}

func (r *PolicyNodeReconciler) managePolicies(ctx context.Context, name string, node *v1.PolicyNode, syncErrs []v1.PolicyNodeSyncError) error {
	var errBuilder multierror.Builder
	reconcileCount := 0
	grs, err := r.decoder.DecodeResources(node.Spec.Resources...)
	if err != nil {
		return errors.Wrapf(err, "could not process policynode: %q", node.GetName())
	}
	for _, gvk := range r.toSync {
		declaredInstances := grs[gvk]
		decorateAsManaged(declaredInstances, node)
		allDeclaredVersions := allVersionNames(grs, gvk.GroupKind())

		actualInstances, err := r.cache.UnstructuredList(gvk, name)
		if err != nil {
			errBuilder.Add(errors.Wrapf(err, "failed to list from policy controller for %q", gvk))
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
	if err := r.setPolicyNodeStatus(ctx, node, syncErrs); err != nil {
		errBuilder.Add(errors.Wrapf(err, "failed to set status for %q", name))
		metrics.ErrTotal.WithLabelValues(node.GetName(), node.GroupVersionKind().Kind, "update").Inc()
		r.recorder.Eventf(node, corev1.EventTypeWarning, "StatusUpdateFailed",
			"failed to update policy node status: %q", err)
	}
	if errBuilder.Len() == 0 && reconcileCount > 0 && len(syncErrs) == 0 {
		r.recorder.Eventf(node, corev1.EventTypeNormal, "ReconcileComplete",
			"policy node was successfully reconciled: %d changes", reconcileCount)
	}
	return errBuilder.Build()
}

// TODO(sbochins): consolidate common functionality with decorateAsClusterManaged.
func decorateAsManaged(declaredInstances []*unstructured.Unstructured, node *v1.PolicyNode) {
	for _, decl := range declaredInstances {
		decl.SetNamespace(node.GetName())
		a := decl.GetAnnotations()
		if a == nil {
			a = map[string]string{}
		}
		// Annotate the resource with the current version token.
		a[v1.SyncTokenAnnotationKey] = node.Spec.ImportToken
		// Annotate the resource as Nomos managed.
		a[v1.ResourceManagementKey] = v1.ResourceManagementValue
		decl.SetAnnotations(a)
	}
}

func (r *PolicyNodeReconciler) setPolicyNodeStatus(ctx context.Context, node *v1.PolicyNode,
	errs []v1.PolicyNodeSyncError) error {
	freshSyncToken := node.Status.SyncToken == node.Spec.ImportToken
	if node.Status.SyncState.IsSynced() && freshSyncToken && len(errs) == 0 {
		glog.Infof("Status for PolicyNode %q is already up-to-date.", node.Name)
		return nil
	}

	updateFn := func(obj runtime.Object) (runtime.Object, error) {
		newPN := obj.(*v1.PolicyNode)
		newPN.Status.SyncToken = node.Spec.ImportToken
		newPN.Status.SyncTime = now()
		newPN.Status.SyncErrors = errs
		if len(errs) > 0 {
			newPN.Status.SyncState = v1.StateError
		} else {
			newPN.Status.SyncState = v1.StateSynced
		}
		newPN.SetGroupVersionKind(kinds.PolicyNode())
		return newPN, nil
	}
	_, err := r.client.UpdateStatus(ctx, node, updateFn)
	return err
}

// NewSyncError returns a PolicyNodeSyncError corresponding to the given error and action
func NewSyncError(name string, gvk schema.GroupVersionKind, err error) v1.PolicyNodeSyncError {
	return v1.PolicyNodeSyncError{
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
		ns := diff.Declared.GetNamespace()
		if err := r.applier.ApplyNamespace(ns, diff.Declared, diff.Actual); err != nil {
			metrics.ErrTotal.WithLabelValues(ns, diff.Declared.GetKind(), "patch").Inc()
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
			metrics.ErrTotal.WithLabelValues(toDelete.GetNamespace(), toDelete.GetKind(), "delete").Inc()
			return false, errors.Wrapf(err, "could not delete resource %q", diff.Name)
		}
	default:
		panic(fmt.Errorf("programmatic error, unhandled syncer diff type: %v", t))
	}
	return true, nil
}

func asNamespace(policyNode *v1.PolicyNode) *corev1.Namespace {
	return withPolicyNodeMeta(&corev1.Namespace{}, policyNode)
}

func withPolicyNodeMeta(namespace *corev1.Namespace, policyNode *v1.PolicyNode) *corev1.Namespace {
	namespace.SetGroupVersionKind(kinds.Namespace())
	// Mark the namespace as supporting the management of hierarchical quota.
	labels := labeling.ManageQuota.AddDeepCopy(policyNode.Labels)
	namespace.Labels = labels
	if as := policyNode.Annotations; as == nil {
		namespace.Annotations = map[string]string{}
	} else {
		namespace.Annotations = policyNode.Annotations
	}
	namespace.Annotations[v1.ResourceManagementKey] = v1.ResourceManagementValue
	namespace.Name = policyNode.Name
	namespace.SetGroupVersionKind(kinds.Namespace())
	return namespace
}

func (r *PolicyNodeReconciler) createNamespace(ctx context.Context, policyNode *v1.PolicyNode) error {
	namespace := asNamespace(policyNode)
	if err := r.client.Create(ctx, namespace); err != nil {
		metrics.ErrTotal.WithLabelValues(namespace.GetName(), namespace.GroupVersionKind().Kind, "create").Inc()
		r.recorder.Eventf(policyNode, corev1.EventTypeWarning, "NamespaceCreateFailed",
			"failed to create matching namespace for policyspace: %q", err)
		return errors.Wrapf(err, "failed to create namespace %q", policyNode.Name)
	}
	return nil
}

func (r *PolicyNodeReconciler) updateNamespace(ctx context.Context, policyNode *v1.PolicyNode) error {
	glog.V(1).Infof("Namespace %q declared in a policy node, updating", policyNode.Name)

	namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: policyNode.Name}}
	namespace.SetGroupVersionKind(kinds.Namespace())
	updateFn := func(obj runtime.Object) (runtime.Object, error) {
		return withPolicyNodeMeta(obj.(*corev1.Namespace), policyNode), nil
	}
	if _, err := r.client.Update(ctx, namespace, updateFn); err != nil {
		metrics.ErrTotal.WithLabelValues(namespace.GetName(), namespace.GroupVersionKind().Kind, "update").Inc()
		r.recorder.Eventf(policyNode, corev1.EventTypeWarning, "NamespaceUpdateFailed",
			"failed to update matching namespace for policyspace: %q", err)
		return errors.Wrapf(err, "failed to update namespace %q", policyNode.Name)
	}
	return nil
}

func (r *PolicyNodeReconciler) deleteNamespace(ctx context.Context, namespace *corev1.Namespace) error {
	glog.V(1).Infof("Namespace %q not declared in a policy node, removing", namespace.GetName())
	if err := r.client.Delete(ctx, namespace); err != nil {
		metrics.ErrTotal.WithLabelValues(namespace.GetName(), namespace.GroupVersionKind().Kind, "delete").Inc()
		return errors.Wrapf(err, "failed to delete namespace %q", namespace.GetName())
	}
	return nil
}

func (r *PolicyNodeReconciler) handlePolicyspace(ctx context.Context, policyNode *v1.PolicyNode) error {
	namespace := asNamespace(policyNode)
	if err := r.client.Delete(ctx, namespace); err != nil {
		metrics.ErrTotal.WithLabelValues("", namespace.GroupVersionKind().Kind, "delete").Inc()
		r.recorder.Eventf(policyNode, corev1.EventTypeWarning, "NamespaceDeleteFailed",
			"failed to delete matching namespace for policyspace: %v", err)
		return errors.Wrapf(err, "failed to delete policyspace %q", policyNode.Name)
	}
	r.recorder.Event(policyNode, corev1.EventTypeNormal, "NamespaceDeleted",
		"removed matching namespace for policyspace")
	return nil
}
