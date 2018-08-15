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
// Package monitor contains the controller for monitoring the state of Nomos on a cluster.
package monitor

import (
	policyhierarchyscheme "github.com/google/nomos/clientgen/policyhierarchy/scheme"
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/monitor/args"
	"github.com/google/nomos/pkg/monitor/clusterpolicy"
	"github.com/google/nomos/pkg/monitor/policynode"
	"github.com/google/nomos/pkg/monitor/state"
	"github.com/google/nomos/pkg/syncer/hierarchy"
	"github.com/kubernetes-sigs/kubebuilder/pkg/inject/run"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/scheme"
)

// Controller is controller for watching Nomos CRDs and exporting metrics about them.
type Controller struct {
	injectArgs   args.InjectArgs
	clusterState *state.ClusterState
}

func NewController(injectArgs args.InjectArgs) *Controller {
	return &Controller{injectArgs, state.NewClusterState()}
}

// Start registers sub controllers and starts them along with their informers.
func (c *Controller) Start(runArgs run.RunArguments) error {
	policyhierarchyscheme.AddToScheme(scheme.Scheme)

	c.injectArgs.ControllerManager.AddController(clusterpolicy.NewController(c.injectArgs, c.clusterState))
	c.injectArgs.ControllerManager.AddController(policynode.NewController(c.injectArgs, c.clusterState))

	c.injectArgs.ControllerManager.RunInformersAndControllers(runArgs)
	return c.initialize()
}

// initialize fetches and populates initial cluster state data.
func (c *Controller) initialize() error {
	// Process the clusterpolicy.
	cpLister := c.injectArgs.Informers.Nomos().V1().ClusterPolicies().Lister()
	cp, err := cpLister.Get(policyhierarchy_v1.ClusterPolicyName)
	if err != nil {
		return errors.Wrap(err, "failed to look up clusterpolicy for monitoring")
	}
	if err = c.clusterState.ProcessClusterPolicy(cp); err != nil {
		return err
	}
	// Process the policynode ancestries.
	pnLister := c.injectArgs.Informers.Nomos().V1().PolicyNodes().Lister()
	policies, err := pnLister.List(labels.Everything())
	if err != nil {
		return errors.Wrap(err, "failed to list all policynodes for monitoring")
	}
	ancestries, err := hierarchy.BuildAncestries(policies)
	if err != nil {
		return errors.Wrap(err, "failed to build ancestries for monitoring")
	}
	for _, ancestry := range ancestries {
		if err := c.clusterState.ProcessPolicyNode(ancestry); err != nil {
			return err
		}
	}
	return nil
}
