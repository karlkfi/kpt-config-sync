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
	// OriginLabelKey is the key for a label that we use to indicate that the resoruce is being
	// managed by the nomos system.
	OriginLabelKey = "nomos-managed"
	// OriginLabelValue is the value we set for OriginLabelKey
	OriginLabelValue = "true"
)

// NewOriginLabel creates a new map with the label.
func NewOriginLabel() map[string]string {
	return map[string]string{
		OriginLabelKey: OriginLabelValue,
	}
}

// WithOriginLabel creates a copy of the map with the new label. This is useful if we are adding
// to a value that we do not want to mutate.
func WithOriginLabel(m map[string]string) map[string]string {
	ret := NewOriginLabel()
	for k, v := range m {
		ret[k] = v
	}
	return ret
}

// NewOriginSelector returns a selector that will select items managed by nomos.
func NewOriginSelector() labels.Selector {
	return labels.Set(NewOriginLabel()).AsSelector()
}

// HasOriginLabel will return true if the given object metadata has been labeled by this package
func HasOriginLabel(objectMeta meta_v1.ObjectMeta) bool {
	if objectMeta.Labels == nil {
		return false
	}
	return objectMeta.Labels[OriginLabelKey] == OriginLabelValue
}

// ObjectHasOriginLabel will return true if the given object has been labeled by this package
func ObjectHasOriginLabel(object meta_v1.Object) bool {
	if object.GetLabels() == nil {
		return false
	}
	return object.GetLabels()[OriginLabelKey] == OriginLabelValue
}
