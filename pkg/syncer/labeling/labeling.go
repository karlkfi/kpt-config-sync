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

package labeling

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// Labels that are used to indicate that the resource is part of the nomos install.
const (
	// TODO(125862145): use policyhierarchy.GroupName below as part of naming cleanup
	ConfigManagementSystemKey   = "configmanagement.gke.io/system"
	ConfigManagementSystemValue = "true"
)

// Labels that Nomos uses to determine which namespaces to enforce hierarchical quota limits on.
const (
	// TODO(125862145): use policyhierarchy.GroupName below s part of naming cleanup
	ConfigManagementQuotaKey   = "configmanagement.gke.io/quota"
	ConfigManagementQuotaValue = "true"
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
func (l *Label) IsSet(object metav1.Object) bool {
	objectLabels := object.GetLabels()
	if objectLabels == nil {
		return false
	}
	return objectLabels[l.key] == l.value
}

// ManageQuota is the label indicating that Nomos manages hierarchical quota in this namespace.
var ManageQuota = Label{
	key:   ConfigManagementQuotaKey,
	value: ConfigManagementQuotaValue,
}

// ConfigManagementSystem indicates that this resource is part of the nomos install.
var ConfigManagementSystem = Label{
	key:   ConfigManagementSystemKey,
	value: ConfigManagementSystemValue,
}
