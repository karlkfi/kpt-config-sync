/*
Copyright 2017 The Stolos Authors.
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

// ClusterPolicySyncerInterface defines the interface for a syncer that consumes ClusterPolicy
type ClusterPolicySyncerInterface interface {
	// OnCreate notifies the syncer to handle a creation event. On startup, this will be called with
	// a create event for each existing ClusterPolicy resource. Once all existing resources
	// have passed through a create event, all future creates will correspond to an actual ClusterPolicy
	// creation event.
	OnCreate(node *policyhierarchy_v1.ClusterPolicy) error

	// OnUpdate notifies the syncer to handle an update event.  This will be triggered if a ClusterPolicy
	// is changed.  Additionally, the informer will periodically resync and send an OnUpdate event for
	// each existing ClusterPolicy resource.  This can be detected by a matching resourceVersion for old and new.
	OnUpdate(old *policyhierarchy_v1.ClusterPolicy, new *policyhierarchy_v1.ClusterPolicy) error

	// OnDelete notifies the syncer to handle a delete event
	OnDelete(node *policyhierarchy_v1.ClusterPolicy) error
}
