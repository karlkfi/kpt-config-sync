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

// Package clusterpolicycontroller handles syncing ClusterPolicy data.
package clusterpolicycontroller

import (
	"time"

	"github.com/golang/glog"
	typed_v1 "github.com/google/nomos/clientgen/apis/typed/nomos/v1"
	policyhierarchy_lister "github.com/google/nomos/clientgen/listers/nomos/v1"
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/nomos/v1"
	"github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/client/object"
	"github.com/google/nomos/pkg/syncer/args"
	"github.com/google/nomos/pkg/syncer/comparator"
	"github.com/google/nomos/pkg/syncer/labeling"
	"github.com/google/nomos/pkg/syncer/metrics"
	"github.com/google/nomos/pkg/syncer/multierror"
	"github.com/google/nomos/pkg/util/clusterpolicy"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/eventhandlers"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/informers"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/types"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
)

const (
	controllerName        = "nomos-cluster-controller"
	reconcileMetricsLabel = "cluster-reconcile"
)

// ClusterPolicyController syncs native Kubernetes resources with the policy
// data in ClusterPolicies.
type ClusterPolicyController struct {
	client   typed_v1.NomosV1Interface
	lister   policyhierarchy_lister.ClusterPolicyLister
	modules  []Module
	recorder record.EventRecorder
}

// informerProvider is here to reduce some amount of redundancy with registering informer providers
// with the controller manager.
type informerProvider struct {
	instance         meta_v1.Object
	informerProvider informers.InformerProvider
}

// NewController creates a new controller.GenericController.
func NewController(
	injectArgs args.InjectArgs,
	modules []Module) *controller.GenericController {
	informer := injectArgs.Informers.Nomos().V1().ClusterPolicies()
	clusterPolicyController := &ClusterPolicyController{
		client:   injectArgs.Clientset.NomosV1(),
		lister:   informer.Lister(),
		modules:  modules,
		recorder: injectArgs.CreateRecorder(controllerName),
	}

	genericController := &controller.GenericController{
		Name:             controllerName,
		InformerRegistry: injectArgs.ControllerManager,
		Reconcile:        clusterPolicyController.reconcile,
	}

	informerProviders := []informerProvider{
		{&policyhierarchy_v1.ClusterPolicy{}, informer},
	}

	for _, m := range modules {
		informerProviders = append(informerProviders, informerProvider{
			instance:         m.Instance(),
			informerProvider: m.InformerProvider(),
		})
	}

	for _, ip := range informerProviders {
		err := injectArgs.ControllerManager.AddInformerProvider(ip.instance, ip.informerProvider)
		if err != nil {
			panic(errors.Wrap(err, "programmer error while adding informer to controller manager"))
		}
	}

	err := genericController.WatchTransformationOf(&policyhierarchy_v1.ClusterPolicy{}, eventhandlers.MapToSelf)
	if err != nil {
		panic(errors.Wrap(err, "programmer error while adding WatchInstanceOf for ClusterPolicies"))
	}

	for _, m := range modules {
		err := genericController.WatchTransformationOf(m.Instance(), mapToClusterPolicy)
		if err != nil {
			panic(errors.Wrapf(err, "programmer error while adding WatchControllerOf for module %s", m.Name()))
		}
	}

	return genericController
}

// reconcile handles ClusterPolicy changes and updates relevant resources
// managed by clusterpolicycontroller.Modules based on those changes.
func (s *ClusterPolicyController) reconcile(k types.ReconcileKey) error {
	metrics.EventTimes.WithLabelValues(reconcileMetricsLabel).Set(float64(time.Now().Unix()))
	timer := prometheus.NewTimer(metrics.ClusterReconcileDuration.WithLabelValues())
	defer timer.ObserveDuration()

	name := k.Name
	cp, err := s.lister.Get(name)
	if err != nil {
		return errors.Wrapf(err, "failed to look up clusterpolicy, %s, for reconciliation", name)
	}

	if name != policyhierarchy_v1.ClusterPolicyName {
		s.recorder.Eventf(cp, core_v1.EventTypeWarning, "InvalidClusterPolicy",
			"ClusterPolicy resource has invalid name %s", name)
		glog.Warningf("ClusterPolicy resource has invalid name %s", name)
		// Return nil since we don't want kubebuilder to queue a retry for this object.
		return nil
	}
	return s.managePolicies(cp)
}

func (s *ClusterPolicyController) managePolicies(cp *policyhierarchy_v1.ClusterPolicy) error {
	var syncErrs []policyhierarchy_v1.ClusterPolicySyncError
	errBuilder := multierror.NewBuilder()
	reconcileCount := 0
	for _, module := range s.modules {
		declaredInstances := module.Extract(cp)

		for idx, decl := range declaredInstances {
			// Identify the ClusterPolicy that is managing this resource.
			blockOwnerDeletion := true
			controller := true
			declaredInstances[idx].SetOwnerReferences([]meta_v1.OwnerReference{
				{
					APIVersion:         policyhierarchy_v1.SchemeGroupVersion.String(),
					Kind:               "ClusterPolicy",
					Name:               cp.Name,
					UID:                cp.UID,
					BlockOwnerDeletion: &blockOwnerDeletion,
					Controller:         &controller,
				},
			})
			// Label the ClusterPolicy resources as nomos-managed.
			declaredInstances[idx].SetLabels(labeling.ManageResource.AddDeepCopy(decl.GetLabels()))
		}

		// Only include nomos-managed resources as current resources. Otherwise, we would end up deleting resources not managed
		// by Nomos.
		actualInstances, err := module.ActionSpec().List("", labeling.ManageResource.Selector())
		if err != nil {
			errBuilder.Add(errors.Wrapf(err, "failed to list from policy controller for %s", module.Name()))
			syncErrs = append(syncErrs, NewSyncError(cp.Name, module.ActionSpec(), err))
			continue
		}

		diffs := comparator.Compare(module.Equal, declaredInstances, object.RuntimeToMeta(actualInstances))
		for _, diff := range diffs {
			if err := execute(diff, module.ActionSpec()); err != nil {
				errBuilder.Add(err)
				syncErrs = append(syncErrs, NewSyncError(cp.Name, module.ActionSpec(), err))
			} else {
				reconcileCount++
			}
		}
	}
	if err := s.setClusterPolicyStatus(cp, syncErrs); err != nil {
		errBuilder.Add(errors.Wrapf(err, "failed to set status for %s", cp.Name))
		s.recorder.Eventf(cp, core_v1.EventTypeWarning, "StatusUpdateFailed",
			"failed to update ClusterPolicy status: %s", err)
	}
	if errBuilder.Len() == 0 && reconcileCount > 0 {
		s.recorder.Eventf(cp, core_v1.EventTypeNormal, "ReconcileComplete",
			"ClusterPolicy was successfully reconciled: %d changes", reconcileCount)
	}
	return errBuilder.Build()
}

func (s *ClusterPolicyController) setClusterPolicyStatus(cp *policyhierarchy_v1.ClusterPolicy, errs []policyhierarchy_v1.ClusterPolicySyncError) error {
	if cp.Status.SyncState.IsSynced() && len(errs) == 0 {
		glog.Infof("Status for ClusterPolicy %s is already up-to-date.", cp.Name)
		return nil
	}
	// TODO(ekitson): Use UpdateStatus() when our minimum supported k8s version is 1.11.
	updateCB := func(old runtime.Object) (runtime.Object, error) {
		oldCP := old.(*policyhierarchy_v1.ClusterPolicy)
		newCP := oldCP.DeepCopy()
		newCP.Status.SyncToken = cp.Spec.ImportToken
		newCP.Status.SyncTime = meta_v1.Now()
		newCP.Status.SyncErrors = errs
		if len(errs) > 0 {
			newCP.Status.SyncState = policyhierarchy_v1.StateError
		} else {
			newCP.Status.SyncState = policyhierarchy_v1.StateSynced
		}
		return newCP, nil
	}
	ua := action.NewReflectiveUpdateAction(
		"", cp.Name, updateCB, clusterpolicy.NewActionSpec(s.client, s.lister))
	return ua.Execute()
}

func NewSyncError(name string, spec *action.ReflectiveActionSpec, err error) policyhierarchy_v1.ClusterPolicySyncError {
	return policyhierarchy_v1.ClusterPolicySyncError{
		ResourceName: name,
		ResourceKind: spec.Resource,
		ResourceAPI:  spec.GroupVersion.String(),
		ErrorMessage: err.Error(),
	}
}

// execute will execute an action based on the Diff.
func execute(diff *comparator.Diff, spec *action.ReflectiveActionSpec) error {
	var act action.Interface
	switch diff.Type {
	case comparator.Add, comparator.Update:
		// generate upsert action
		act = action.NewReflectiveUpsertAction(
			"", diff.Declared.GetName(), diff.Declared.(runtime.Object), spec)
	case comparator.Delete:
		// generate delete action
		act = action.NewReflectiveDeleteAction("", diff.Actual.GetName(), spec)
	}
	if err := act.Execute(); err != nil {
		glog.V(4).Infof("Operation %s failed", act)
		return errors.Wrapf(err, "failed to execute action: %s", act)
	}
	return nil
}

func mapToClusterPolicy(obj interface{}) string {
	return policyhierarchy_v1.ClusterPolicyName
}
