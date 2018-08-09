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

	"github.com/google/go-cmp/cmp"
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/syncer/hierarchy"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SyncState enumerates the current status between a Nomos custom resource and its core resources.
type SyncState string

const (
	stateUnknown SyncState = ""
	stateSynced  SyncState = "synced"
	stateStale   SyncState = "stale"
	stateError   SyncState = "error"
)

func clusterPolicyState(cp *policyhierarchy_v1.ClusterPolicy) SyncState {
	if len(cp.Status.SyncErrors) > 0 {
		return stateError
	}
	if cp.Spec.ImportToken == cp.Status.SyncToken {
		return stateSynced
	}
	return stateStale
}

func policyNodeState(ancestry hierarchy.Ancestry) SyncState {
	pn := ancestry.Node()
	if len(pn.Status.SyncErrors) > 0 {
		return stateError
	}
	if cmp.Equal(pn.Status.SyncTokens, ancestry.TokenMap()) {
		return stateSynced
	}
	return stateStale
}

// ClusterState maintains the status of imports and syncs at the cluster level and exports them as
// Prometheus metrics.
type ClusterState struct {
	mux        sync.Mutex
	lastImport time.Time
	lastSync   time.Time
	syncStates map[string]SyncState
}

func NewClusterState() *ClusterState {
	return &ClusterState{
		syncStates: map[string]SyncState{},
	}
}

// DeletePolicy removes the ClusterPolicy or PolicyNode with the given name if it is present.
func (c *ClusterState) DeletePolicy(name string) {
	c.mux.Lock()
	defer c.mux.Unlock()

	delete(c.syncStates, name)
}

// ProcessClusterPolicy updates the ClusterState with the current status of the ClusterPolicy.
func (c *ClusterState) ProcessClusterPolicy(cp *policyhierarchy_v1.ClusterPolicy) error {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.updateTimes(cp.Spec.ImportTime, cp.Status.SyncTime)
	newState := clusterPolicyState(cp)
	c.recordLatency(cp.Name, newState, cp.Spec.ImportTime, cp.Status.SyncTime)

	if err := c.updateState(cp.Name, newState); err != nil {
		return errors.Wrap(err, "while processing cluster policy state")
	}
	return nil
}

// ProcessClusterPolicy updates the ClusterState with the current status of the PolicyNode.
func (c *ClusterState) ProcessPolicyNode(ancestry hierarchy.Ancestry) error {
	c.mux.Lock()
	defer c.mux.Unlock()

	pn := ancestry.Node()
	c.updateTimes(pn.Spec.ImportTime, pn.Status.SyncTime)
	newState := policyNodeState(ancestry)
	c.recordLatency(pn.Name, newState, pn.Spec.ImportTime, pn.Status.SyncTime)

	if err := c.updateState(pn.Name, newState); err != nil {
		return errors.Wrap(err, "while processing policy node state")
	}
	return nil
}

func (c *ClusterState) recordLatency(name string, newState SyncState, importTime, syncTime metav1.Time) {
	oldState := c.syncStates[name]
	if oldState == stateSynced || newState != stateSynced {
		return
	}
	Metrics.SyncLatency.Observe(float64(syncTime.Unix() - importTime.Unix()))
}

func (c *ClusterState) updateState(name string, newState SyncState) error {
	oldState := c.syncStates[name]
	if oldState == newState {
		return nil
	}
	newMetric, err := Metrics.ClusterNodes.GetMetricWithLabelValues(string(newState))
	if err != nil {
		return err
	}
	newMetric.Inc()

	if oldState != stateUnknown {
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
