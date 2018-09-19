/*
Copyright 2017 The Nomos Authors.
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

// Package clusterpolicy contains the controller for monitoring Nomos ClusterPolicies.
package clusterpolicy

import (
	"github.com/golang/glog"
	policyhierarchylister "github.com/google/nomos/clientgen/listers/policyhierarchy/v1"
	policyhierarchyv1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/monitor/args"
	"github.com/google/nomos/pkg/monitor/state"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/eventhandlers"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/types"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	controllerName = "nomos-monitor-clusterpolicy-controller"
)

// Controller responds to changes to ClusterPolicies by updating its ClusterState.
type Controller struct {
	lister policyhierarchylister.ClusterPolicyLister
	state  *state.ClusterState
}

// NewController creates a new controller.GenericController.
func NewController(injectArgs args.InjectArgs, state *state.ClusterState) *controller.GenericController {
	informer := injectArgs.Informers.Nomos().V1().ClusterPolicies()
	cpController := &Controller{informer.Lister(), state}

	genericController := &controller.GenericController{
		Name:             controllerName,
		InformerRegistry: injectArgs.ControllerManager,
		Reconcile:        cpController.reconcile,
	}
	cp := &policyhierarchyv1.ClusterPolicy{}

	if err := injectArgs.ControllerManager.AddInformerProvider(cp, informer); err != nil {
		panic(errors.Wrap(err, "programmer error while adding informer to controller manager"))
	}
	if err := genericController.WatchTransformationOf(cp, eventhandlers.MapToSelf); err != nil {
		panic(errors.Wrap(err, "programmer error while adding WatchInstanceOf for clusterpolicies"))
	}
	return genericController
}

func (c *Controller) reconcile(k types.ReconcileKey) error {
	name := k.Name
	cp, err := c.lister.Get(name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			c.state.DeletePolicy(name)
			return nil
		}
		return errors.Wrapf(err, "failed to look up clusterpolicy %s for monitoring", name)
	}
	if name != policyhierarchyv1.ClusterPolicyName {
		glog.Errorf("clusterpolicy resource has invalid name %q", name)
		// Return nil since we don't want kubebuilder to queue a retry.
		return nil
	}
	err = c.state.ProcessClusterPolicy(cp)
	if err != nil {
		glog.Error(err)
	}
	return nil
}
