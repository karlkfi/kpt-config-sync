/*
Copyright 2017 The CSP Config Management Authors.
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

	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterState maintains the status of imports and syncs at the cluster level and exports them as
// Prometheus metrics.
type ClusterState struct {
	mux        sync.Mutex
	lastImport time.Time
	lastSync   time.Time
	syncStates map[string]v1.ConfigSyncState
}

// NewClusterState returns a new ClusterState.
func NewClusterState() *ClusterState {
	return &ClusterState{
		syncStates: map[string]v1.ConfigSyncState{},
	}
}

// DeleteConfig removes the ClusterConfig or NamespaceConfig with the given name if it is present.
func (c *ClusterState) DeleteConfig(name string) {
	c.mux.Lock()
	defer c.mux.Unlock()

	delete(c.syncStates, name)
}

// ProcessClusterConfig updates the ClusterState with the current status of the ClusterConfig.
func (c *ClusterState) ProcessClusterConfig(cp *v1.ClusterConfig) error {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.updateTimes(cp.Spec.ImportTime, cp.Status.SyncTime)
	c.recordLatency(cp.Name, cp.Status.SyncState, cp.Spec.ImportTime, cp.Status.SyncTime)

	if err := c.updateState(cp.Name, cp.Status.SyncState); err != nil {
		return errors.Wrap(err, "while processing cluster config state")
	}
	return nil
}

// ProcessNamespaceConfig updates the ClusterState with the current status of the NamespaceConfig.
func (c *ClusterState) ProcessNamespaceConfig(pn *v1.NamespaceConfig) error {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.updateTimes(pn.Spec.ImportTime, pn.Status.SyncTime)
	c.recordLatency(pn.Name, pn.Status.SyncState, pn.Spec.ImportTime, pn.Status.SyncTime)

	if err := c.updateState(pn.Name, pn.Status.SyncState); err != nil {
		return errors.Wrap(err, "while processing namespace config state")
	}

	return nil
}

func (c *ClusterState) recordLatency(name string, newState v1.ConfigSyncState, importTime, syncTime metav1.Time) {
	oldState := c.syncStates[name]
	if oldState.IsSynced() || !newState.IsSynced() {
		return
	}
	Metrics.SyncLatency.Observe(float64(syncTime.Unix() - importTime.Unix()))
}

func (c *ClusterState) updateState(name string, newState v1.ConfigSyncState) error {
	oldState := c.syncStates[name]
	if oldState == newState {
		return nil
	}

	Metrics.Configs.WithLabelValues(string(newState)).Inc()
	if oldState != v1.StateUnknown {
		Metrics.Configs.WithLabelValues(string(oldState)).Dec()
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
