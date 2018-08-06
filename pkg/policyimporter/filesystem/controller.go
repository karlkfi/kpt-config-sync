// Reviewed by sunilarora
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

package filesystem

import (
	"path/filepath"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/clientgen/informers/policyhierarchy"
	listers_v1 "github.com/google/nomos/clientgen/listers/policyhierarchy/v1"
	policyhierarchyscheme "github.com/google/nomos/clientgen/policyhierarchy/scheme"
	"github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/client/meta"
	"github.com/google/nomos/pkg/policyimporter"
	"github.com/google/nomos/pkg/policyimporter/actions"
	"github.com/google/nomos/pkg/policyimporter/git"
	"github.com/google/nomos/pkg/util/policynode"
	"github.com/pkg/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

const resync = time.Minute * 15

// Controller is controller for managing Nomos CRDs by importing policies from a filesystem tree.
type Controller struct {
	policyDir           string
	pollPeriod          time.Duration
	parser              *Parser
	differ              *actions.Differ
	informerFactory     policyhierarchy.SharedInformerFactory
	policyNodeLister    listers_v1.PolicyNodeLister
	clusterPolicyLister listers_v1.ClusterPolicyLister
	stopChan            chan struct{}
}

// NewController returns a new Controller.
func NewController(policyDir string, pollPeriod time.Duration, parser *Parser, client meta.Interface, stopChan chan struct{}) *Controller {
	policyhierarchyscheme.AddToScheme(scheme.Scheme)

	informerFactory := policyhierarchy.NewSharedInformerFactory(
		client.PolicyHierarchy(), resync)
	differ := actions.NewDiffer(
		actions.NewFactories(
			client.PolicyHierarchy().NomosV1(),
			informerFactory.Nomos().V1().PolicyNodes().Lister(),
			informerFactory.Nomos().V1().ClusterPolicies().Lister()))

	return &Controller{
		policyDir:           policyDir,
		pollPeriod:          pollPeriod,
		parser:              parser,
		differ:              differ,
		informerFactory:     informerFactory,
		policyNodeLister:    informerFactory.Nomos().V1().PolicyNodes().Lister(),
		clusterPolicyLister: informerFactory.Nomos().V1().ClusterPolicies().Lister(),
		stopChan:            stopChan,
	}
}

// Run runs the controller and blocks until an error occurs or stopChan is closed.
func (c *Controller) Run() error {
	// Start informers
	c.informerFactory.Start(c.stopChan)
	glog.Infof("Waiting for cache to sync")
	synced := c.informerFactory.WaitForCacheSync(c.stopChan)
	for syncType, ok := range synced {
		if !ok {
			elemType := syncType.Elem()
			return errors.Errorf("Failed to sync %s:%s", elemType.PkgPath(), elemType.Name())
		}
	}
	glog.Infof("Caches synced")

	return c.pollDir()
}

func (c *Controller) pollDir() error {
	glog.Infof("Polling policy dir: %s", c.policyDir)

	currentPolicies, err := policynode.ListPolicies(c.policyNodeLister, c.clusterPolicyLister)
	if err != nil {
		return errors.Wrapf(err, "failed to list current policies")
	}

	currentDir := ""
	ticker := time.NewTicker(c.pollPeriod)

	for {
		select {
		case <-ticker.C:
			// Detect whether symlink has changed.
			newDir, err := filepath.EvalSymlinks(c.policyDir)
			if err != nil {
				return errors.Wrapf(err, "failed to resolve policy dir")
			}
			if currentDir == newDir {
				// No new commits, nothing to do.
				continue
			}
			glog.Infof("Resolved policy dir: %s", newDir)

			// Parse filesystem tree into in-memory PolicyNode and ClusterPolicy objects.
			desiredPolicies, err := c.parser.Parse(newDir)
			if err != nil {
				glog.Warningf("Failed to parse: %v", err)
				policyimporter.Metrics.PolicyStates.WithLabelValues("failed").Inc()
				continue
			}

			// Parse the commit hash from the new directory to use as an import token.
			token, err := git.CommitHash(newDir)
			if err != nil {
				glog.Warningf("Failed to parse commit hash: %v", err)
				policyimporter.Metrics.PolicyStates.WithLabelValues("failed").Inc()
				continue
			}

			// Update the import tokens and times for all policy nodes and cluster policy.
			time := meta_v1.Now()
			for n := range desiredPolicies.PolicyNodes {
				pn := desiredPolicies.PolicyNodes[n]
				pn.Spec.ImportToken = token
				pn.Spec.ImportTime = time
				desiredPolicies.PolicyNodes[n] = pn
			}
			desiredPolicies.ClusterPolicy.Spec.ImportToken = token
			desiredPolicies.ClusterPolicy.Spec.ImportTime = time

			// Calculate the sequence of actions needed to transition from current to desired state.
			actions := c.differ.Diff(*currentPolicies, *desiredPolicies)
			if err := applyActions(actions); err != nil {
				glog.Warningf("Failed to apply actions: %v", err)
				policyimporter.Metrics.PolicyStates.WithLabelValues("failed").Inc()
				continue
			}

			currentDir = newDir
			currentPolicies = desiredPolicies
			policyimporter.Metrics.PolicyStates.WithLabelValues("succeeded").Inc()
			policyimporter.Metrics.Nodes.Set(float64(len(desiredPolicies.PolicyNodes)))

		case <-c.stopChan:
			glog.Info("Stop polling")
			return nil
		}
	}
}

func applyActions(actions []action.Interface) error {
	for _, a := range actions {
		if err := a.Execute(); err != nil {
			return err
		}
	}
	return nil
}
