/*
Copyright 2017 The Kubernetes Authors.
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

package syncer

import (
	policyhierarchy_v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"
)

// PolicyNodeSyncerInterface defines the interface for a sycner that consumes PolicyNodes, caller
// must filter for duplicate add events from the initial sync if re-creation is not desired.
type PolicyNodeSyncerInterface interface {
	// InitialSync performs the initial sync for the resource
	InitialSync(node []*policyhierarchy_v1.PolicyNode) error

	// OnCreate notifies the syncer to handle a creation event
	OnCreate(node *policyhierarchy_v1.PolicyNode) error

	// OnUpdate notifies the syncer to handle an update event
	OnUpdate(old *policyhierarchy_v1.PolicyNode, new *policyhierarchy_v1.PolicyNode) error

	// OnDelete notifies the syncer to handle a delete event
	OnDelete(node *policyhierarchy_v1.PolicyNode) error
}
