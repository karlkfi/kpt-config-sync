/*
Copyright 2019 The Nomos Authors.
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

package sync

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GroupKinds returns a set of GroupKinds represented by the slice of Syncs.
func GroupKinds(syncs ...*v1alpha1.Sync) map[schema.GroupKind]bool {
	gks := make(map[schema.GroupKind]bool)
	for _, sync := range syncs {
		for _, g := range sync.Spec.Groups {
			for _, k := range g.Kinds {
				gk := schema.GroupKind{
					Group: g.Group,
					Kind:  k.Kind,
				}
				gks[gk] = true
			}
		}
	}
	return gks
}
