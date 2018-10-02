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
// Reviewed by sunilarora

// Package policyhierarchycontroller defines a kubebuilder controller.GenericController that will
// handle hierarchical policy synchronization.  The goal for this package is to create a common
// controller and enable additional policies to be onboarded by implementing only the minimal
// logic required for hierarchical policy computation and resource reconciliation.
package policyhierarchycontroller

import (
	"flag"
	"time"

	"github.com/golang/glog"
	typedv1 "github.com/google/nomos/clientgen/apis/typed/policyhierarchy/v1"
	policyhierarchylister "github.com/google/nomos/clientgen/listers/policyhierarchy/v1"
	policyhierarchyv1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/client/object"
	"github.com/google/nomos/pkg/syncer/actions"
	"github.com/google/nomos/pkg/syncer/args"
	"github.com/google/nomos/pkg/syncer/comparator"
	"github.com/google/nomos/pkg/syncer/eventprocessor"
	"github.com/google/nomos/pkg/syncer/labeling"
	"github.com/google/nomos/pkg/syncer/metrics"
	"github.com/google/nomos/pkg/syncer/parentindexer"
	"github.com/google/nomos/pkg/util/multierror"
	"github.com/google/nomos/pkg/util/namespaceutil"
	"github.com/google/nomos/pkg/util/policynode"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/informers"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/types"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/record"
)

var (
	hierarchyWarningTimeout = flag.Duration(
		"hierarchyWarningTimeout",
		time.Minute*60,
		"The amount of time that must pass before hierarchy issues are surfaced via warning events")
	hierarchyWarningCount = flag.Int64(
		"hierarchyWarningCount",
		3,
		"The number of times that hierarchy consistency errors must occur to be surfaced via warning events.")
)

const controllerName = "nomos-hierarchical-controller"

// PolicyHieraryController controls policies based on the declarations in the PolicyNodes.
type PolicyHieraryController struct {
	injectArgs        args.InjectArgs
	client            typedv1.NomosV1Interface
	nodeLister        policyhierarchylister.PolicyNodeLister
	namespaceLister   listerscorev1.NamespaceLister
	modules           []Module
	recorder          record.EventRecorder
	hierarchyWarnings *WarningFilter
}

// informerProvider is here to reduce some amount of redunancy with registering informer providers
// with the controller manager.
type informerProvider struct {
	instance         metav1.Object
	informerProvider informers.InformerProvider
}

// NewController creates a new controller.GenericController that will synchronize hierarchical policies.
func NewController(
	injectArgs args.InjectArgs,
	modules []Module) *controller.GenericController {
	policyHierarchyController := &PolicyHieraryController{
		injectArgs:        injectArgs,
		client:            injectArgs.Clientset.NomosV1(),
		nodeLister:        injectArgs.Informers.Nomos().V1().PolicyNodes().Lister(),
		namespaceLister:   injectArgs.KubernetesInformers.Core().V1().Namespaces().Lister(),
		modules:           modules,
		recorder:          injectArgs.CreateRecorder(controllerName),
		hierarchyWarnings: NewWarningFilter(*hierarchyWarningCount, *hierarchyWarningTimeout),
	}
	err := injectArgs.Informers.Nomos().V1().PolicyNodes().Informer().AddIndexers(
		parentindexer.Indexer())
	if err != nil {
		panic(errors.Wrapf(err, "unrecoverable error"))
	}

	genericController := &controller.GenericController{
		Name:             controllerName,
		InformerRegistry: injectArgs.ControllerManager,
		Reconcile:        policyHierarchyController.reconcile,
	}

	informerProviders := []informerProvider{
		{&policyhierarchyv1.PolicyNode{}, injectArgs.Informers.Nomos().V1().PolicyNodes()},
		{&corev1.Namespace{}, injectArgs.KubernetesInformers.Core().V1().Namespaces()},
	}
	for _, module := range modules {
		informerProviders = append(informerProviders, informerProvider{
			instance:         module.Instance(),
			informerProvider: module.InformerProvider(),
		})
	}
	for _, item := range informerProviders {
		err = injectArgs.ControllerManager.AddInformerProvider(item.instance, item.informerProvider)
		if err != nil {
			panic(errors.Wrap(err, "programmer error while adding informer to controller manager"))
		}
	}

	err = genericController.WatchEvents(&policyhierarchyv1.PolicyNode{}, eventprocessor.Factory(
		injectArgs.Informers.Nomos().V1().PolicyNodes().Lister()))
	if err != nil {
		panic(errors.Wrapf(err, "programmer error while adding WatchEvents for policynode"))
	}
	err = genericController.Watch(&corev1.Namespace{})
	if err != nil {
		panic(errors.Wrapf(err, "programmer error while adding Watch for namespace"))
	}

	for _, module := range modules {
		err := genericController.WatchTransformationOf(module.Instance(), objectKeyNamespaceToName)
		if err != nil {
			panic(errors.Wrapf(err, "programmer error while adding Watch for %s", module.Name()))
		}
	}

	return genericController
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
func (s *PolicyHieraryController) getPolicyNodeState(name string) (policyNodeState, *policyhierarchyv1.PolicyNode, error) {
	node, err := s.nodeLister.Get(name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return policyNodeStateNotFound, nil, nil
		}
		glog.Fatalf("Lister returned error other than not found, this should not happen: %v", err)
	}
	s.hierarchyWarnings.Clear(name)

	if namespaceutil.IsReserved(name) {
		return policyNodeStateReserved, node, nil
	}

	switch node.Spec.Type {
	case policyhierarchyv1.Policyspace:
		return policyNodeStatePolicyspace, node, nil
	case policyhierarchyv1.Namespace:
		return policyNodeStateNamespace, node, nil
	case policyhierarchyv1.ReservedNamespace:
		return policyNodeStateReserved, node, nil
	default:
		return policyNodeStateNotFound, nil, errors.Errorf("Invalid node type %q", node.Spec.Type)
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
func (s *PolicyHieraryController) getNamespaceState(name string) (namespaceState, *corev1.Namespace, error) {
	ns, err := s.namespaceLister.Get(name)
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

	glog.Warningf("Namespace %s has invalid management label %s", name, value)
	s.recorder.Eventf(
		ns,
		corev1.EventTypeWarning,
		"InvalidManagmentLabel",
		"Namespace %s has invalid management label %s",
		name, value,
	)
	return namespaceStateExists, ns, nil
}

func (s *PolicyHieraryController) reconcile(k types.ReconcileKey) error {
	name := k.Name
	metrics.EventTimes.WithLabelValues("hierarchy-reconcile").Set(float64(time.Now().Unix()))
	reconcileTimer := prometheus.NewTimer(
		metrics.HierarchicalReconcileDuration.WithLabelValues(name))
	defer reconcileTimer.ObserveDuration()

	if *strictNamespaceSync {
		return s.hardReconcile(name)
	}
	return s.softReconcile(name)
}

func (s *PolicyHieraryController) softReconcile(name string) error {
	pnState, node, pnErr := s.getPolicyNodeState(name)
	if pnErr != nil {
		return pnErr
	}

	nsState, ns, nsErr := s.getNamespaceState(name)
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
			s.warnUndeclaredNamespace(ns)
		case namespaceStateManagePolicies:
			s.warnUndeclaredNamespace(ns)
		case namespaceStateManageFull:
			return s.deleteNamespace(name)
		}

	case policyNodeStateNamespace:
		switch nsState {
		case namespaceStateNotFound:
			if err := s.createNamespace(node); err != nil {
				return err
			}
			return s.managePolicies(name, node)
		case namespaceStateExists:
			s.warnNoLabel(ns)
		case namespaceStateManagePolicies:
			if err := s.updateNamespace(node); err != nil {
				return err
			}
			return s.managePolicies(name, node)
		case namespaceStateManageFull:
			if err := s.upsertNamespace(node); err != nil {
				return err
			}
			return s.managePolicies(name, node)
		}

	case policyNodeStatePolicyspace:
		switch nsState {
		case namespaceStateNotFound: // noop
		case namespaceStateExists:
			s.warnPolicyspaceHasNamespace(ns)
		case namespaceStateManagePolicies:
			s.warnPolicyspaceHasNamespace(ns)
		case namespaceStateManageFull:
			return s.handlePolicyspace(node)
		}

	case policyNodeStateReserved:
		switch nsState {
		case namespaceStateNotFound: // noop
		case namespaceStateExists: // noop
		case namespaceStateManagePolicies:
			s.warnReservedLabel(ns)
		case namespaceStateManageFull:
			s.warnReservedLabel(ns)
		}
	}
	return nil
}

func (s *PolicyHieraryController) warnUndeclaredNamespace(ns *corev1.Namespace) {
	glog.Warningf("namespace %q exists but is not declared in the source of truth", ns.Name)
	s.recorder.Event(
		ns, corev1.EventTypeWarning, "UnmanagedNamespace",
		"namespace is not declared in the source of truth")
}

func (s *PolicyHieraryController) warnPolicyspaceHasNamespace(ns *corev1.Namespace) {
	glog.Warningf("namespace %q exists but is declared as a policyspace in the source of truth", ns.Name)
	s.recorder.Event(
		ns, corev1.EventTypeWarning, "NamespaceInPolicySpace",
		"namespace is declared as a policyspace in the source of truth")
}

func (s *PolicyHieraryController) warnReservedLabel(ns *corev1.Namespace) {
	glog.Warningf("reserved namespace %q has a management label", ns.Name)
	s.recorder.Event(
		ns, corev1.EventTypeWarning, "UnmanagedNamespace",
		"reserved namespace has a management label")
}

func (s *PolicyHieraryController) warnNoLabel(ns *corev1.Namespace) {
	glog.Warningf("namespace %q is declared in the source of truth but does not have a management label", ns.Name)
	s.recorder.Event(
		ns, corev1.EventTypeWarning, "UnmanagedNamespace",
		"namespace is declared in the source of truth but does not have a management label")
}

func (s *PolicyHieraryController) hardReconcile(name string) error {
	if namespaceutil.IsReserved(name) {
		return nil
	}

	node, err := s.nodeLister.Get(name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return s.deleteNamespace(name)
		}
		panic("Lister returned error other than not found, this should not happen")
	}

	switch node.Spec.Type {
	case policyhierarchyv1.Policyspace:
		return s.handlePolicyspace(node)
	case policyhierarchyv1.ReservedNamespace:
		return nil
	}

	if err := s.upsertNamespace(node); err != nil {
		return errors.Wrapf(err, "failed to upsert namespace %s", name)
	}

	return s.managePolicies(name, node)
}

func (s *PolicyHieraryController) managePolicies(name string, node *policyhierarchyv1.PolicyNode) error {
	var syncErrs []policyhierarchyv1.PolicyNodeSyncError
	errBuilder := multierror.NewBuilder()
	reconcileCount := 0
	for _, module := range s.modules {
		declaredInstances := module.Instances(node)

		for _, decl := range declaredInstances {
			decl.SetNamespace(name)
			decl.SetLabels(labeling.ManageResource.AddDeepCopy(decl.GetLabels()))
		}

		actualInstances, err := module.ActionSpec().List(name, labels.Everything())
		if err != nil {
			errBuilder.Add(errors.Wrapf(err, "failed to list from policy controller for %s", module.Name()))
			syncErrs = append(syncErrs, NewSyncError(name, module.ActionSpec(), err))
			continue
		}

		diffs := comparator.Compare(module.Equal, declaredInstances, object.RuntimeToMeta(actualInstances))
		for _, diff := range diffs {
			if err := s.handleDiff(name, module, diff); err != nil {
				errBuilder.Add(err)
				pse := NewSyncError(name, module.ActionSpec(), err)
				pse.ResourceName = diff.Name
				syncErrs = append(syncErrs, pse)
			} else {
				reconcileCount++
			}
		}
	}
	if err := s.setPolicyNodeStatus(node, syncErrs); err != nil {
		errBuilder.Add(errors.Wrapf(err, "failed to set status for %s", name))
		s.recorder.Eventf(node, corev1.EventTypeWarning, "StatusUpdateFailed",
			"failed to update policy node status: %s", err)
	}
	if errBuilder.Len() == 0 && reconcileCount > 0 {
		s.recorder.Eventf(node, corev1.EventTypeNormal, "ReconcileComplete",
			"policy node was successfully reconciled: %d changes", reconcileCount)
	}
	return errBuilder.Build()
}

func (s *PolicyHieraryController) setPolicyNodeStatus(node *policyhierarchyv1.PolicyNode, errs []policyhierarchyv1.PolicyNodeSyncError) error {
	if node.Status.SyncState.IsSynced() && len(errs) == 0 {
		glog.Infof("Status for PolicyNode %s is already up-to-date.", node.Name)
		return nil
	}
	// TODO(ekitson): Use UpdateStatus() when our minimum supported k8s version is 1.11.
	updateCB := func(old runtime.Object) (runtime.Object, error) {
		oldPN := old.(*policyhierarchyv1.PolicyNode)
		newPN := oldPN.DeepCopy()
		newPN.Status.SyncTokens = map[string]string{node.Name: node.Spec.ImportToken}
		newPN.Status.SyncTime = metav1.Now()
		newPN.Status.SyncErrors = errs
		if len(errs) > 0 {
			newPN.Status.SyncState = policyhierarchyv1.StateError
		} else {
			newPN.Status.SyncState = policyhierarchyv1.StateSynced
		}
		return newPN, nil
	}
	ua := action.NewReflectiveUpdateAction(
		"", node.Name, updateCB, policynode.NewActionSpec(s.client, s.nodeLister))
	return ua.Execute()
}

// NewSyncError returns a PolicyNodeSyncError corresponding to the given error and action
func NewSyncError(name string, spec *action.ReflectiveActionSpec, err error) policyhierarchyv1.PolicyNodeSyncError {
	return policyhierarchyv1.PolicyNodeSyncError{
		SourceName:   name,
		ResourceKind: spec.Resource,
		ResourceAPI:  spec.GroupVersion.String(),
		ErrorMessage: err.Error(),
	}
}

func (s *PolicyHieraryController) handleDiff(namespace string, module Module, diff *comparator.Diff) error {
	var act action.Interface
	switch diff.Type {
	case comparator.Add:
		fallthrough
	case comparator.Update:
		act = action.NewReflectiveUpsertAction(
			namespace, diff.Declared.GetName(), diff.Declared.(runtime.Object), module.ActionSpec())
	case comparator.Delete:
		act = action.NewReflectiveDeleteAction(
			namespace, diff.Actual.GetName(), module.ActionSpec())
	}
	if err := act.Execute(); err != nil {
		return errors.Wrapf(err, "failed to execute %s", act)
	}
	return nil
}

func (s *PolicyHieraryController) namespaceValue(policyNode *policyhierarchyv1.PolicyNode) *corev1.Namespace {
	labels := labeling.ManageAll.AddDeepCopy(policyNode.Labels)
	labels[policyhierarchyv1.ParentLabelKey] = policyNode.Spec.Parent
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        policyNode.Name,
			Labels:      labels,
			Annotations: policyNode.Annotations,
		},
	}
}

func (s *PolicyHieraryController) createNamespace(policyNode *policyhierarchyv1.PolicyNode) error {
	act := actions.NewNamespaceCreateAction(
		s.namespaceValue(policyNode),
		s.injectArgs.KubernetesClientSet,
		s.namespaceLister)
	if err := act.Execute(); err != nil {
		s.recorder.Eventf(policyNode, corev1.EventTypeWarning, "NamespaceCreateFailed",
			"failed to create matching namespace for policyspace: %s", err)
		return errors.Wrapf(err, "failed to execute create action for %s", act)
	}
	return nil
}

func (s *PolicyHieraryController) upsertNamespace(policyNode *policyhierarchyv1.PolicyNode) error {
	glog.V(1).Infof("Namespace %s declared in a policy node, upserting", policyNode.Name)
	act := actions.NewNamespaceUpsertAction(
		s.namespaceValue(policyNode),
		s.injectArgs.KubernetesClientSet,
		s.namespaceLister)
	if err := act.Execute(); err != nil {
		s.recorder.Eventf(policyNode, corev1.EventTypeWarning, "NamespaceUpsertFailed",
			"failed to upsert matching namespace for policyspace: %s", err)
		return errors.Wrapf(err, "failed to execute upsert action for %s", act)
	}
	return nil
}

// updateNamespace is used for updating the parent label on a namespace where we manage policy values
// This is used since we can't update all the labels on the namespace.
func (s *PolicyHieraryController) updateNamespace(policyNode *policyhierarchyv1.PolicyNode) error {
	labels := map[string]string{policyhierarchyv1.ParentLabelKey: policyNode.Spec.Parent}
	act := actions.NewNamespaceUpdateAction(
		policyNode.Name,
		actions.SetNamespaceLabelsFunc(labels),
		s.injectArgs.KubernetesClientSet,
		s.namespaceLister)
	if err := act.Execute(); err != nil {
		s.recorder.Eventf(policyNode, corev1.EventTypeWarning, "NamespaceUpdateFailed",
			"failed to update matching namespace for policyspace: %s", err)
		return errors.Wrapf(err, "failed to execute update action for %s", act)
	}
	return nil
}

func (s *PolicyHieraryController) deleteNamespace(name string) error {
	glog.V(1).Infof("Namespace %s not declared in a policy node, removing", name)
	act := actions.NewNamespaceDeleteAction(name, s.injectArgs.KubernetesClientSet, s.namespaceLister)
	if err := act.Execute(); err != nil {
		return errors.Wrapf(err, "failed to execute delete action for %s", act)
	}
	return nil
}

func (s *PolicyHieraryController) handlePolicyspace(policyNode *policyhierarchyv1.PolicyNode) error {
	act := actions.NewNamespaceDeleteAction(policyNode.Name, s.injectArgs.KubernetesClientSet, s.namespaceLister)
	if err := act.Execute(); err != nil {
		s.recorder.Eventf(policyNode, corev1.EventTypeWarning, "NamespaceDeleteFailed",
			"failed to delete matching namespace for policyspace: %s", err)
		return errors.Wrapf(err, "failed to execute delete action for policyspace %s", act)
	}
	s.recorder.Event(policyNode, corev1.EventTypeNormal, "NamespaceDeleted",
		"removed matching namespace for policyspace")
	return nil
}
