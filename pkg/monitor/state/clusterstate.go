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

// Package state contains information about the state of Nomos on a cluster.
package state

import (
	"sync"
	"time"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterState maintains the status of imports and syncs at the cluster level and exports them as
// Prometheus metrics.
type ClusterState struct {
	mux        sync.Mutex
	lastImport time.Time
	lastSync   time.Time
	syncStates map[string]v1.PolicySyncState
}

// NewClusterState returns a new ClusterState.
func NewClusterState() *ClusterState {
	return &ClusterState{
		syncStates: map[string]v1.PolicySyncState{},
	}
}

// DeletePolicy removes the ClusterPolicy or PolicyNode with the given name if it is present.
func (c *ClusterState) DeletePolicy(name string) {
	c.mux.Lock()
	defer c.mux.Unlock()

	delete(c.syncStates, name)
}

// ProcessClusterPolicy updates the ClusterState with the current status of the ClusterPolicy.
func (c *ClusterState) ProcessClusterPolicy(cp *v1.ClusterPolicy) error {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.updateTimes(cp.Spec.ImportTime, cp.Status.SyncTime)
	c.recordLatency(cp.Name, cp.Status.SyncState, cp.Spec.ImportTime, cp.Status.SyncTime)

	if err := c.updateState(cp.Name, cp.Status.SyncState); err != nil {
		return errors.Wrap(err, "while processing cluster policy state")
	}
	return nil
}

// ProcessPolicyNode updates the ClusterState with the current status of the PolicyNode.
func (c *ClusterState) ProcessPolicyNode(pn *v1.PolicyNode) error {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.updateTimes(pn.Spec.ImportTime, pn.Status.SyncTime)

	return nil
}

func (c *ClusterState) recordLatency(name string, newState v1.PolicySyncState, importTime, syncTime metav1.Time) {
	oldState := c.syncStates[name]
	if oldState.IsSynced() || !newState.IsSynced() {
		return
	}
	Metrics.SyncLatency.Observe(float64(syncTime.Unix() - importTime.Unix()))
}

func (c *ClusterState) updateState(name string, newState v1.PolicySyncState) error {
	oldState := c.syncStates[name]
	if oldState == newState {
		return nil
	}
	newMetric, err := Metrics.ClusterNodes.GetMetricWithLabelValues(string(newState))
	if err != nil {
		return err
	}
	newMetric.Inc()

	if oldState != v1.StateUnknown {
		oldMetric, err := Metrics.ClusterNodes.GetMetricWithLabelValues(string(oldState))
		if err != nil {
			return err
		}
		oldMetric.Dec()
	}
	c.syncStates[name] = newState
	return nil
}

func (c *ClusterState) updateTimes(importTime, syncTime metav1.Time) {
	if importTime.After(c.lastImport) {
		c.lastImport = importTime.Time
		Metrics.LastImport.Set(float64(importTime.Unix()))
	}
	if syncTime.After(c.lastSync) {
		c.lastSync = syncTime.Time
		Metrics.LastSync.Set(float64(syncTime.Unix()))
	}
}
