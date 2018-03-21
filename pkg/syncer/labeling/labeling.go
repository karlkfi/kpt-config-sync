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

// OriginLabel is defined to make it easy to set the label if none is given
// on the object.
var OriginLabel = map[string]string{OriginLabelKey: OriginLabelValue}

// AddOriginLabelToMap adds the origin label to a map containing labels for an object. The map
// passed to the function is also returned for convenience.
func AddOriginLabelToMap(labelMap map[string]string) map[string]string {
	if labelMap == nil {
		labelMap = map[string]string{}
	}
	labelMap[OriginLabelKey] = OriginLabelValue
	return labelMap
}

// AddOriginLabel adds the provenance (managed by nomos) label to an object's metadata.
func AddOriginLabel(objectMeta *meta_v1.ObjectMeta) {
	objectMeta.Labels = AddOriginLabelToMap(objectMeta.Labels)
}

// NewOriginSelector returns a selector that will select items managed by nomos.
func NewOriginSelector() labels.Selector {
	return labels.Set(OriginLabel).AsSelector()
}

// HasOriginLabel will return true if the given object metadata has been labeled by this package
func HasOriginLabel(objectMeta meta_v1.ObjectMeta) bool {
	if objectMeta.Labels == nil {
		return false
	}
	return objectMeta.Labels[OriginLabelKey] == OriginLabelValue
}
