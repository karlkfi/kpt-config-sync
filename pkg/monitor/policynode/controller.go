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

// Package policynode contains the controller for monitoring Nomos PolicyNodes.
package policynode

import (
	"github.com/golang/glog"
	policyhierarchylister "github.com/google/nomos/clientgen/listers/policyhierarchy/v1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/monitor/args"
	"github.com/google/nomos/pkg/monitor/state"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/eventhandlers"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/types"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	controllerName = "nomos-monitor-policynode-controller"
)

// Controller responds to changes to PolicyNodes by updating its ClusterState.
type Controller struct {
	lister policyhierarchylister.PolicyNodeLister
	state  *state.ClusterState
}

// NewController creates a new controller.GenericController.
func NewController(injectArgs args.InjectArgs, state *state.ClusterState) *controller.GenericController {
	informer := injectArgs.Informers.Nomos().V1().PolicyNodes()
	pnController := &Controller{
		informer.Lister(),
		state,
	}

	genericController := &controller.GenericController{
		Name:             controllerName,
		InformerRegistry: injectArgs.ControllerManager,
		Reconcile:        pnController.reconcile,
	}
	pn := &v1.PolicyNode{}

	if err := injectArgs.ControllerManager.AddInformerProvider(pn, informer); err != nil {
		panic(errors.Wrap(err, "programmer error while adding informer to controller manager"))
	}
	if err := genericController.WatchTransformationOf(pn, eventhandlers.MapToSelf); err != nil {
		panic(errors.Wrap(err, "programmer error while adding WatchInstanceOf for policynodes"))
	}
	return genericController
}

func (c *Controller) reconcile(k types.ReconcileKey) error {
	name := k.Name
	node, err := c.lister.Get(name)
	if err == nil {
		return c.state.ProcessPolicyNode(node)
	}
	switch {
	case apierrors.IsNotFound(err):
		c.state.DeletePolicy(name)
		return nil
	default:
		glog.Errorf("Failed to fetch policy node for %q.", name)
	}
	return errors.Wrapf(err, "failed to look up policynode %s for monitoring", name)
}
