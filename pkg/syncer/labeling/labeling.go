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

// Labels that indicate the resource was created by Nomos
const (
	// ManagedLabelKey is the key for a label that we use to indicate that the resource is being
	// managed by the nomos system. Resources that do not have this label will not be touched by
	// Nomos.
	ManagedLabelKey = "nomos-managed"
	// True indicates that the label key's state is now active.
	True = "true"
)

// NewManagedLabel creates a new map with the label.
func NewManagedLabel() map[string]string {
	return map[string]string{
		ManagedLabelKey: True,
	}
}

// AddManagedDeepCopy adds the managed label to a copy of the given map. The original map is not
// modified.
func AddManagedDeepCopy(m map[string]string) map[string]string {
	ret := map[string]string{ManagedLabelKey: True}
	for k, v := range m {
		ret[k] = v
	}
	return ret
}

// AddManaged adds the managed label to a map.
func AddManaged(m map[string]string) {
	m[ManagedLabelKey] = True
}

// NewManagedSelector returns a selector that will select items managed by nomos.
func NewManagedSelector() labels.Selector {
	return labels.Set(NewManagedLabel()).AsSelector()
}

// HasManagedLabel will return true if the given object metadata has been labeled by this package
func HasManagedLabel(objectMeta meta_v1.ObjectMeta) bool {
	if objectMeta.Labels == nil {
		return false
	}
	return objectMeta.Labels[ManagedLabelKey] == True
}

// ObjectHasManagedLabel will return true if the given object has been labeled by this package
func ObjectHasManagedLabel(object meta_v1.Object) bool {
	if object.GetLabels() == nil {
		return false
	}
	return object.GetLabels()[ManagedLabelKey] == True
}
