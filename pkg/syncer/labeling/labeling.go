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
// Reviewed by sunilarora

package labeling

import (
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// Labels that Nomos uses for determining how to manage namespaces.
const (
	// ManagementKey is the key for a label that Nomos uses to track which namespaces are under Nomos
	// management and what sort of management Nomos will perform.
	ManagementKey = "nomos.dev/namespace-management"

	// Policies indicates that Nomos will manage policies for the namespace.
	Policies = "policies"

	// All indicates that Nomos will manage policies and namespace lifecycle for the given namespace.
	Full = "full"
)

// Labels that Nomos uses for determining how to manage resources other than namespaces.
const (
	// ResourceManagementKey indicates that Nomos manages the lifecycle for the resource (this label
	// does not apply to namespaces.
	ResourceManagementKey = "nomos.dev/managed"
)

// Label helps manage applying labels to objects.
type Label struct {
	key   string
	value string
}

// New returns a new map with the labels.
func (l *Label) New() map[string]string {
	return map[string]string{l.key: l.value}
}

// AddTo adds the label to a map.
func (l *Label) AddTo(m map[string]string) {
	m[l.key] = l.value
}

// AddDeepCopy creates a copy of the provided map then adds the label. The original map is not modified.
func (l *Label) AddDeepCopy(m map[string]string) map[string]string {
	return labels.Merge(m, l.New())
}

// Selector returns a selector for the label.
func (l *Label) Selector() labels.Selector {
	return labels.Set(l.New()).AsSelector()
}

// IsSet returns true if the label is set on the object with matching value.
func (l *Label) IsSet(object meta_v1.Object) bool {
	objectLabels := object.GetLabels()
	if objectLabels == nil {
		return false
	}
	return objectLabels[l.key] == l.value
}

// ManagePolicies is the label indicating that Nomos owns management of policies.
var ManagePolicies = Label{
	key:   ManagementKey,
	value: Policies,
}

// ManageAll is the label indicating that Nomos owns management of both policy and namespace lifecycle.
var ManageAll = Label{
	key:   ManagementKey,
	value: Full,
}

// ManageResource is the label indicating that Nomos manages the lifecycle for a resource.
var ManageResource = Label{
	key:   ResourceManagementKey,
	value: Full,
}
