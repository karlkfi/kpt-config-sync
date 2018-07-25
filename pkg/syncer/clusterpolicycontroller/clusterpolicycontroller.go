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
	policyhierarchy_lister "github.com/google/nomos/clientgen/listers/policyhierarchy/v1"
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/client/object"
	"github.com/google/nomos/pkg/syncer/args"
	"github.com/google/nomos/pkg/syncer/comparator"
	"github.com/google/nomos/pkg/syncer/labeling"
	"github.com/google/nomos/pkg/syncer/metrics"
	"github.com/google/nomos/pkg/syncer/multierror"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/eventhandlers"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/informers"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/types"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const reconcileMetricsLabel = "cluster-reconcile"

// ClusterPolicyController syncs native Kubernetes resources with the policy
// data in ClusterPolicies.
type ClusterPolicyController struct {
	lister  policyhierarchy_lister.ClusterPolicyLister
	modules []Module
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
		lister:  informer.Lister(),
		modules: modules,
	}

	genericController := &controller.GenericController{
		Name:             "nomos-cluster-controller",
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
	if name != policyhierarchy_v1.ClusterPolicyName {
		// TODO(briantkennedy): we may want to generate a kubernetes event for this scenario.
		glog.Warningf("ClusterPolicy resource has invalid name %s", name)
		// Return nil since we don't want kubebuilder to queue a retry for this object.
		return nil
	}

	cp, err := s.lister.Get(name)
	if err != nil {
		return errors.Wrapf(err, "failed to look up clusterpolicy, %s, for reconciliation", name)
	}
	return s.managePolicies(cp)
}

func (s *ClusterPolicyController) managePolicies(cp *policyhierarchy_v1.ClusterPolicy) error {
	var syncErrs []policyhierarchy_v1.ClusterSyncError
	errBuilder := multierror.NewBuilder()
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
			syncErrs = append(syncErrs, NewClusterSyncError(cp.Name, module.ActionSpec(), err))
			continue
		}

		diffs := comparator.Compare(module.Equal, declaredInstances, object.RuntimeToMeta(actualInstances))
		for _, diff := range diffs {
			if err := execute(diff, module.ActionSpec()); err != nil {
				errBuilder.Add(err)
				syncErrs = append(syncErrs, NewClusterSyncError(cp.Name, module.ActionSpec(), err))
			}
		}
	}
	setClusterPolicyStatus(cp, syncErrs)

	if errBuilder.Len() != 0 {
		glog.Warningf("reconcile encountered %d errors", errBuilder.Len())
	}
	return errBuilder.Build()
}

func setClusterPolicyStatus(cp *policyhierarchy_v1.ClusterPolicy, errs []policyhierarchy_v1.ClusterSyncError) {
	// TODO(ekitson): Use .DeepCopy() to avoid updating the shared cache version of the clusterpolicy.
	cp.Status.SyncToken = cp.Spec.ImportToken
	cp.Status.SyncTime = meta_v1.Now()
	cp.Status.SyncErrors = errs
}

func NewClusterSyncError(name string, spec *action.ReflectiveActionSpec, err error) policyhierarchy_v1.ClusterSyncError {
	gv := schema.GroupVersion{Group: spec.Group, Version: spec.Version}
	return policyhierarchy_v1.ClusterSyncError{
		ResourceName: name,
		ResourceKind: spec.Resource,
		ResourceAPI:  gv.String(),
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
