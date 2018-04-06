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

package policyhierarchycontroller

import (
	"github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/syncer/hierarchy"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/informers"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Module is a type specific implementation for hierarchical synchronization.
// Each module is responsible for synchronizing only one resource type.
type Module interface {
	// Name is the name of the module, this will mainly be used for logging
	// purposes.
	Name() string

	// Equal must implement a function that determines if the content
	// fields of the two objects are equivalent. This should only consider
	// annotations if they are overloaded to hold additional data for the
	// object.
	Equal(meta_v1.Object, meta_v1.Object) bool

	// NewAggregatedNode returns a new aggregated node object that will
	// perform hierarchy aggregation operations for the type.
	NewAggregatedNode() hierarchy.AggregatedNode

	// Instance returns an instance of the type that this module is going
	// to be synchronizing. Since it operates on API types, they should all
	// satisfy the meta_v1.Object interface.
	Instance() meta_v1.Object

	// InformerProvider returns an informer provider for the controlled
	// resource type.
	InformerProvider() informers.InformerProvider

	// ActionSpec returns the spec for the API type that this module will
	// be synchronizing. This should correspond to a spec for the same type
	// that Instance returns.
	ActionSpec() *action.ReflectiveActionSpec
}
