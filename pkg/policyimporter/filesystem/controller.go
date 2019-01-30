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

package filesystem

import (
	"flag"
	"path/filepath"
	"reflect"
	"time"

	"github.com/golang/glog"
	policyhierarchyscheme "github.com/google/nomos/clientgen/apis/scheme"
	"github.com/google/nomos/clientgen/informer"
	listersv1 "github.com/google/nomos/clientgen/listers/policyhierarchy/v1"
	listersv1alpha1 "github.com/google/nomos/clientgen/listers/policyhierarchy/v1alpha1"
	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/client/meta"
	"github.com/google/nomos/pkg/policyimporter"
	"github.com/google/nomos/pkg/policyimporter/actions"
	"github.com/google/nomos/pkg/policyimporter/git"
	"github.com/google/nomos/pkg/util/policynode"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes/scheme"
)

const resync = time.Minute * 15

var syncDeleteMaxWait = flag.Duration("sync_delete_max_wait", 15*time.Second,
	"Number of seconds to wait for Syncer to acknowledge Sync deletion")

// Controller is controller for managing Nomos CRDs by importing policies from a filesystem tree.
type Controller struct {
	policyDir           string
	pollPeriod          time.Duration
	parser              *Parser
	differ              *actions.Differ
	discoveryClient     discovery.ServerResourcesInterface
	informerFactory     informer.SharedInformerFactory
	policyNodeLister    listersv1.PolicyNodeLister
	clusterPolicyLister listersv1.ClusterPolicyLister
	syncLister          listersv1alpha1.SyncLister
	stopChan            chan struct{}
	client              meta.Interface
}

// NewController returns a new Controller.
func NewController(policyDir string, pollPeriod time.Duration, parser *Parser, client meta.Interface, stopChan chan struct{}) *Controller {
	policyhierarchyscheme.AddToScheme(scheme.Scheme)

	informerFactory := informer.NewSharedInformerFactory(
		client.PolicyHierarchy(), resync)
	differ := actions.NewDiffer(
		actions.NewFactories(
			client.PolicyHierarchy().NomosV1(),
			client.PolicyHierarchy().NomosV1alpha1(),
			informerFactory.Nomos().V1().PolicyNodes().Lister(),
			informerFactory.Nomos().V1().ClusterPolicies().Lister(),
			informerFactory.Nomos().V1alpha1().Syncs().Lister()))

	return &Controller{
		policyDir:           policyDir,
		pollPeriod:          pollPeriod,
		parser:              parser,
		differ:              differ,
		discoveryClient:     client.Kubernetes().Discovery(),
		informerFactory:     informerFactory,
		policyNodeLister:    informerFactory.Nomos().V1().PolicyNodes().Lister(),
		clusterPolicyLister: informerFactory.Nomos().V1().ClusterPolicies().Lister(),
		syncLister:          informerFactory.Nomos().V1alpha1().Syncs().Lister(),
		stopChan:            stopChan,
		client:              client,
	}
}

// Run runs the controller and blocks until an error occurs or stopChan is closed.
//
// Each iteration of the loop does the following:
//   * Checks for updates to the filesystem that stores policy source of truth.
//   * When there are updates, parses the filesystem into AllPolicies, an in-memory
//     representation of desired policies.
//   * Gets the policies currently stored in Kubernetes API server.
//   * Compares current and desired policies.
//   * Writes updates to make current match desired.
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

	c.pollDir()
	return nil
}

func (c *Controller) pollDir() {
	currentDir := ""
	ticker := time.NewTicker(c.pollPeriod)

	for {
		select {
		case <-ticker.C:
			// Detect whether symlink has changed.
			newDir, err := filepath.EvalSymlinks(c.policyDir)
			if err != nil {
				glog.Errorf("failed to resolve policydir: %v", err)
				policyimporter.Metrics.PolicyStates.WithLabelValues("failed").Inc()
				continue
			}
			if currentDir == newDir {
				// No new commits, nothing to do.
				continue
			}
			glog.Infof("Resolved policy dir: %s. Polling policy dir: %s", newDir, c.policyDir)

			currentPolicies, err := policynode.ListPolicies(c.policyNodeLister, c.clusterPolicyLister, c.syncLister)
			if err != nil {
				glog.Errorf("failed to list current policies: %v", err)
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

			loadTime := time.Now()
			// Parse filesystem tree into in-memory PolicyNode and ClusterPolicy objects.
			desiredPolicies, err := c.parser.Parse(newDir, token, loadTime)
			if err != nil {
				glog.Warningf("Failed to parse: %v", err)
				policyimporter.Metrics.PolicyStates.WithLabelValues("failed").Inc()
				continue
			}

			// Update the SyncState for all policy nodes and cluster policy.
			for n := range desiredPolicies.PolicyNodes {
				pn := desiredPolicies.PolicyNodes[n]
				pn.Status.SyncState = v1.StateStale
				desiredPolicies.PolicyNodes[n] = pn
			}
			desiredPolicies.ClusterPolicy.Status.SyncState = v1.StateStale

			if err := c.updatePolicies(currentPolicies, desiredPolicies); err != nil {
				glog.Warningf("Failed to apply actions: %v", err)
				policyimporter.Metrics.PolicyStates.WithLabelValues("failed").Inc()
				continue
			}

			currentDir = newDir
			policyimporter.Metrics.PolicyStates.WithLabelValues("succeeded").Inc()
			policyimporter.Metrics.Nodes.Set(float64(len(desiredPolicies.PolicyNodes)))

		case <-c.stopChan:
			glog.Info("Stop polling")
			return
		}
	}
}

// updatePolicies calculates and applies the actions needed to go from current to desired.
// The order of actions is as follows:
//   1. Delete Syncs. This includes any Syncs that are deleted outright, as well as any Syncs that
//      are present in both current and desired, but which lose one or more SyncVersions in the
//      transition.
//   2. Apply PolicyNode and ClusterPolicy updates.
//   3. Apply remaining Sync updates.
//
// This careful ordering matters in the case where both a Sync and a Resource of the same type are
// deleted in the same commit. The desired outcome is that the resource is not deleted, so we delete
// the Sync first. That way, Syncer stops listening to updates for that type before the resource is
// deleted from policies.
//
// If the same resource and Sync are added again in a subsequent commit, the ordering ensures that
// the resource is restored in policy before the Syncer starts managing that type.
func (c *Controller) updatePolicies(current, desired *v1.AllPolicies) error {
	if err := c.syncDeletes(current, desired); err != nil {
		return err
	}
	if err := c.syncReductions(current, desired); err != nil {
		return err
	}
	// Calculate the sequence of actions needed to transition from current to desired state.
	a := c.differ.Diff(*current, *desired)
	return applyActions(a)
}

// syncDeletes gets a list of Syncs to be deleted from the differ. It deletes them, and waits for
// the deletes to be finalized.
//
// Failure to wait for the deletes to finalize could lead to a Sync delete racing with actions for
// resources of the Sync's specified type. In that case, some resources actions might apply before
// the Sync is deleted, and some after. Once the Sync is gone, Syncer will stop watching resources
// of that type. So by not waiting, Syncer's response to those actions would be non-deterministic.
func (c *Controller) syncDeletes(current, desired *v1.AllPolicies) error {
	syncDeletes := c.differ.SyncDeletes(current.Syncs, desired.Syncs)
	for _, sd := range syncDeletes {
		if err := c.client.PolicyHierarchy().NomosV1alpha1().Syncs().Delete(sd, &metav1.DeleteOptions{}); err != nil {
			return err
		}
	}
	for _, sd := range syncDeletes {
		exists := true
		deadline := time.Now().Add(*syncDeleteMaxWait)
		for exists {
			if time.Now().After(deadline) {
				return errors.Errorf("timeout waiting for deleted Sync %s to finalize", sd)
			}

			glog.V(1).Infof("Polling for deleted Sync %s", sd)
			_, err := c.client.PolicyHierarchy().NomosV1alpha1().Syncs().Get(sd, metav1.GetOptions{})
			if err == nil {
				time.Sleep(50 * time.Millisecond)
			} else {
				if apierrors.IsNotFound(err) {
					exists = false
				} else {
					return errors.Wrapf(err, "while polling for deleted Sync %s: ", sd)
				}
			}
		}
	}
	return nil
}

// syncReductions gets a list of Syncs to be updated during the Delete Syncs phase
func (c *Controller) syncReductions(current, desired *v1.AllPolicies) error {
	toReduce := c.differ.SyncReductions(current.Syncs, desired.Syncs)
	for _, sr := range toReduce {
		if _, err := c.client.PolicyHierarchy().NomosV1alpha1().Syncs().Update(&sr); err != nil {
			return err
		}
	}
	for _, sr := range toReduce {
		expectedStatus := v1alpha1.SyncStatus{}
		for _, g := range sr.Spec.Groups {
			for _, k := range g.Kinds {
				for _, v := range k.Versions {
					expectedStatus.GroupVersionKinds =
						append(expectedStatus.GroupVersionKinds,
							v1alpha1.SyncGroupVersionKindStatus{Group: g.Group, Kind: k.Kind, Version: v.Version})
				}
			}
		}
		deadline := time.Now().Add(*syncDeleteMaxWait)
		var actualStatus *v1alpha1.SyncStatus
		for !statusEqual(&expectedStatus, actualStatus) {
			if time.Now().After(deadline) {
				return errors.Errorf("timeout waiting for statuses to converge for Sync %q", sr.Name)
			}
			time.Sleep(500 * time.Millisecond)
			actualSync, err := c.client.PolicyHierarchy().NomosV1alpha1().Syncs().Get(sr.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			actualStatus = &actualSync.Status
		}
	}
	return nil
}

func statusEqual(expected, actual *v1alpha1.SyncStatus) bool {
	expectedSet := make(map[v1alpha1.SyncGroupVersionKindStatus]struct{})
	for _, e := range expected.GroupVersionKinds {
		expectedSet[e] = struct{}{}
	}
	actualSet := make(map[v1alpha1.SyncGroupVersionKindStatus]struct{})
	for _, e := range actual.GroupVersionKinds {
		actualSet[e] = struct{}{}
	}
	return reflect.DeepEqual(expectedSet, actualSet)
}

func applyActions(actions []action.Interface) error {
	for _, a := range actions {
		if err := a.Execute(); err != nil {
			return err
		}
	}
	return nil
}
