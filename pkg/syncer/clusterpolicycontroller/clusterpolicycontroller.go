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

// Package clusterpolicycontroller handles syncing ClusterPolicy data.
package clusterpolicycontroller

import (
	"time"

	policyhierarchy_lister "github.com/google/nomos/clientgen/listers/policyhierarchy/v1"
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/client/object"
	"github.com/google/nomos/pkg/syncer/args"
	"github.com/google/nomos/pkg/syncer/comparator"
	"github.com/google/nomos/pkg/syncer/labeling"
	"github.com/google/nomos/pkg/syncer/metrics"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/eventhandlers"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/informers"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/types"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
		err := genericController.WatchControllerOf(m.Instance(), eventhandlers.Path{clusterPolicyController.ownerLookup})
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
	clusterPolicy, err := s.lister.Get(name)
	if err != nil {
		return errors.Wrapf(err, "failed to look up clusterpolicy, %s, for reconciliation", name)
	}

	// We return the first error we encounter during module processing.
	var moduleErr error
	for _, module := range s.modules {
		declaredInstances := module.Extract(clusterPolicy)
		for _, decl := range declaredInstances {
			// Identify the ClusterPolicy that is managing this resource.
			blockOwnerDeletion := true
			controller := true
			decl.SetOwnerReferences([]meta_v1.OwnerReference{
				{
					APIVersion:         policyhierarchy_v1.SchemeGroupVersion.String(),
					Kind:               "ClusterPolicy",
					Name:               clusterPolicy.Name,
					UID:                clusterPolicy.UID,
					BlockOwnerDeletion: &blockOwnerDeletion,
					Controller:         &controller,
				},
			})
			// Label the ClusterPolicy resources as nomos-managed.
			decl.SetLabels(labeling.WithOriginLabel(decl.GetLabels()))
		}

		// Only include nomos-managed resources as current resources. Otherwise, we would end up deleting resources not managed
		// by Nomos.
		actualInstances, err := module.ActionSpec().List("", labeling.NewOriginSelector())
		if err != nil {
			moduleErr = errors.Wrapf(err, "failed to list from policy controller for %s", module.Name())
			continue
		}

		diffs := comparator.Compare(module.Equal, declaredInstances, object.RuntimeToMeta(actualInstances))
		for _, diff := range diffs {
			if err := execute(diff, module.ActionSpec()); err != nil {
				if moduleErr != nil {
					moduleErr = err
				}
				break
			}
		}
	}
	return moduleErr
}

func (s *ClusterPolicyController) ownerLookup(k types.ReconcileKey) (interface{}, error) {
	return s.lister.Get(k.Name)
}

// execute will execute an action based on the Diff.
func execute(diff *comparator.Diff, spec *action.ReflectiveActionSpec) error {
	switch diff.Type {
	case comparator.Add:
		fallthrough
	case comparator.Update:
		// generate upsert action
		a := action.NewReflectiveUpsertAction(
			"", diff.Declared.GetName(), diff.Declared.(runtime.Object), spec)
		if err := a.Execute(); err != nil {
			return errors.Wrapf(err, "failed to execute delete action: %s", a)
		}
	case comparator.Delete:
		// generate delete action
		a := action.NewReflectiveDeleteAction("", diff.Actual.GetName(), spec)
		if err := a.Execute(); err != nil {
			return errors.Wrapf(err, "failed to execute delete action: %s", a)
		}
	}
	return nil
}
