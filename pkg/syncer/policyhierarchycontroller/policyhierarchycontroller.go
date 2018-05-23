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
// handle hierarchical policy synchonization.  The goal for this package is to create a common
// controller and enable additional policies to be onboarded by implementing only the minimal
// logic required for hierarchical policy computation and resource reconciliation.
package policyhierarchycontroller

import (
	"time"

	"github.com/golang/glog"
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/client/object"
	"github.com/google/nomos/pkg/syncer/actions"
	"github.com/google/nomos/pkg/syncer/args"
	"github.com/google/nomos/pkg/syncer/comparator"
	"github.com/google/nomos/pkg/syncer/eventprocessor"
	"github.com/google/nomos/pkg/syncer/hierarchy"
	"github.com/google/nomos/pkg/syncer/labeling"
	"github.com/google/nomos/pkg/syncer/metrics"
	"github.com/google/nomos/pkg/syncer/multierror"
	"github.com/google/nomos/pkg/syncer/parentindexer"
	"github.com/google/nomos/pkg/util/namespaceutil"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/informers"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/types"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	core_v1 "k8s.io/api/core/v1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	listers_core_v1 "k8s.io/client-go/listers/core/v1"
)

// PolicyHieraryController controls policies based on the declarations in the PolicyNodes.
type PolicyHieraryController struct {
	injectArgs      args.InjectArgs
	namespaceLister listers_core_v1.NamespaceLister
	hierarchy       hierarchy.Interface
	modules         []Module
}

// informerProvider is here to reduce some amount of redunancy with registering informer providers
// with the controller manager.
type informerProvider struct {
	instance         meta_v1.Object
	informerProvider informers.InformerProvider
}

// NewController creates a new controller.GenericController that will synchronize hierarchical policies.
func NewController(
	injectArgs args.InjectArgs,
	modules []Module) *controller.GenericController {
	policyHierarchyController := &PolicyHieraryController{
		injectArgs:      injectArgs,
		namespaceLister: injectArgs.KubernetesInformers.Core().V1().Namespaces().Lister(),
		modules:         modules,
		hierarchy:       hierarchy.New(injectArgs.Informers.Nomos().V1().PolicyNodes()),
	}
	err := injectArgs.Informers.Nomos().V1().PolicyNodes().Informer().AddIndexers(
		parentindexer.Indexer())
	if err != nil {
		panic(errors.Wrapf(err, "unrecoverable error"))
	}

	genericController := &controller.GenericController{
		Name:             "nomos-hierarchical-controller",
		InformerRegistry: injectArgs.ControllerManager,
		Reconcile:        policyHierarchyController.reconcile,
	}

	informerProviders := []informerProvider{
		{&policyhierarchy_v1.PolicyNode{}, injectArgs.Informers.Nomos().V1().PolicyNodes()},
		{&core_v1.Namespace{}, injectArgs.KubernetesInformers.Core().V1().Namespaces()},
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

	err = genericController.WatchEvents(&policyhierarchy_v1.PolicyNode{}, eventprocessor.Factory(
		injectArgs.Informers.Nomos().V1().PolicyNodes()))
	if err != nil {
		panic(errors.Wrapf(err, "programmer error while adding WatchEvents for policynode"))
	}
	err = genericController.Watch(&core_v1.Namespace{})
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

// getPolicyNodeState normalizes the state of the policy node while retrieving ancestry.
func (s *PolicyHieraryController) getPolicyNodeState(name string) (policyNodeState, hierarchy.Ancestry, error) {
	ancestry, err := s.hierarchy.Ancestry(name)
	if err != nil {
		switch {
		case hierarchy.IsNotFoundError(err):
			return policyNodeStateNotFound, nil, nil
		case hierarchy.IsIncompleteHierarchyError(err):
			return policyNodeStateNotFound, ancestry, errors.Wrapf(err, "incomplete ancestry")
		default:
			return policyNodeStateNotFound, ancestry, errors.Wrapf(err, "unhandled error while fetching ancestry")
		}
	}

	if namespaceutil.IsReserved(name) {
		return policyNodeStateReserved, ancestry, nil
	}

	switch ancestry.Node().Spec.Type {
	case policyhierarchy_v1.Policyspace:
		return policyNodeStatePolicyspace, ancestry, nil
	case policyhierarchy_v1.Namespace:
		return policyNodeStateNamespace, ancestry, nil
	case policyhierarchy_v1.ReservedNamespace:
		return policyNodeStateReserved, ancestry, nil
	default:
		return policyNodeStateNotFound, ancestry, errors.Errorf("Invalid node type %q for ancestry %v", ancestry.Node().Spec.Type, ancestry)
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
func (s *PolicyHieraryController) getNamespaceState(name string) (namespaceState, *core_v1.Namespace, error) {
	ns, err := s.namespaceLister.Get(name)
	if err != nil {
		if api_errors.IsNotFound(err) {
			return namespaceStateNotFound, nil, nil
		}
		return namespaceStateNotFound, nil, errors.Wrapf(err, "got unhandled lister error")
	}

	if labeling.ManageAll.IsSet(ns) {
		return namespaceStateManageFull, ns, nil
	}

	if labeling.ManagePolicies.IsSet(ns) {
		return namespaceStateManagePolicies, ns, nil
	}

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
	pnState, ancestry, pnErr := s.getPolicyNodeState(name)
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
			if err := s.createNamespace(ancestry.Node()); err != nil {
				return err
			}
			return s.managePolicies(name, ancestry)
		case namespaceStateExists:
			s.warnNoLabel(ns)
		case namespaceStateManagePolicies, namespaceStateManageFull:
			return s.managePolicies(name, ancestry)
		}

	case policyNodeStatePolicyspace:
		switch nsState {
		case namespaceStateNotFound: // noop
		case namespaceStateExists:
			s.warnPolicyspaceHasNamespace(ns)
		case namespaceStateManagePolicies:
			s.warnPolicyspaceHasNamespace(ns)
		case namespaceStateManageFull:
			return s.deleteNamespace(name)
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

// TODO: emit warning events for the warn* functions
func (s *PolicyHieraryController) warnUndeclaredNamespace(ns *core_v1.Namespace) {
	glog.Warningf("namespace %q exists but is not declared in the source of truth", ns.Name)
}

func (s *PolicyHieraryController) warnPolicyspaceHasNamespace(ns *core_v1.Namespace) {
	glog.Warningf("namespace %q exists but is declared as a policyspace in the source of truth", ns.Name)
}

func (s *PolicyHieraryController) warnReservedLabel(ns *core_v1.Namespace) {
	glog.Warningf("reserved namespace %q has a management label", ns.Name)
}

func (s *PolicyHieraryController) warnNoLabel(ns *core_v1.Namespace) {
	glog.Warningf("namespace %q is declared in the source of truth but does not have a management label", ns.Name)
}

func (s *PolicyHieraryController) hardReconcile(name string) error {
	if namespaceutil.IsReserved(name) {
		return nil
	}

	ancestry, err := s.hierarchy.Ancestry(name)
	if err != nil {
		switch {
		case hierarchy.IsNotFoundError(err):
			return s.deleteNamespace(name)
		case hierarchy.IsIncompleteHierarchyError(err):
			return errors.Wrapf(err, "incomplete ancestry")
		default:
			return errors.Wrapf(err, "unhandled error while fetching ancestry")
		}
	}

	switch ancestry.Node().Spec.Type {
	case policyhierarchy_v1.Policyspace:
		return s.handlePolicyspace(name)
	case policyhierarchy_v1.ReservedNamespace:
		return nil
	}

	if err := s.upsertNamespace(ancestry.Node()); err != nil {
		return errors.Wrapf(err, "failed to upsert namespace %s", name)
	}

	return s.managePolicies(name, ancestry)
}

func (s *PolicyHieraryController) managePolicies(name string, ancestry hierarchy.Ancestry) error {
	errBuilder := multierror.NewBuilder()
	for _, module := range s.modules {
		declaredInstances := ancestry.Aggregate(module.NewAggregatedNode)

		for _, decl := range declaredInstances {
			decl.SetNamespace(name)
			decl.SetLabels(labeling.ManageResource.AddDeepCopy(decl.GetLabels()))
		}

		actualInstances, err := module.ActionSpec().List(name, labels.Everything())
		if err != nil {
			return errors.Wrapf(err, "failed to list from policy controller for %s", module.Name())
		}

		diffs := comparator.Compare(module.Equal, declaredInstances, object.RuntimeToMeta(actualInstances))
		for _, diff := range diffs {
			if err := s.handleDiff(name, module, diff); err != nil {
				errBuilder.Add(err)
			}
		}
	}
	return nil
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

func (s *PolicyHieraryController) createNamespace(policyNode *policyhierarchy_v1.PolicyNode) error {
	labels := labeling.ManageAll.New()
	labels[policyhierarchy_v1.ParentLabelKey] = policyNode.Spec.Parent
	act := actions.NewNamespaceCreateAction(
		policyNode.Name,
		"",
		labels,
		s.injectArgs.KubernetesClientSet,
		s.namespaceLister)
	if err := act.Execute(); err != nil {
		return errors.Wrapf(err, "failed to execute upsert action for %s", act)
	}
	return nil
}

func (s *PolicyHieraryController) upsertNamespace(policyNode *policyhierarchy_v1.PolicyNode) error {
	glog.V(1).Infof("Namespace %s declared in a policy node, upserting", policyNode.Name)

	labels := labeling.ManageAll.New()
	labels[policyhierarchy_v1.ParentLabelKey] = policyNode.Spec.Parent
	act := actions.NewNamespaceUpsertAction(
		policyNode.Name,
		policyNode.UID,
		labels,
		s.injectArgs.KubernetesClientSet,
		s.namespaceLister)
	if err := act.Execute(); err != nil {
		return errors.Wrapf(err, "failed to execute upsert action for %s", act)
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

func (s *PolicyHieraryController) handlePolicyspace(name string) error {
	act := actions.NewNamespaceDeleteAction(name, s.injectArgs.KubernetesClientSet, s.namespaceLister)
	if err := act.Execute(); err != nil {
		return errors.Wrapf(err, "failed to execute delete action for policyspace %s", act)
	}
	return nil
}
