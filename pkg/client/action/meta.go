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

package action

import (
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// isSubset returns true if subset is a subset of set.  If subset is empty, it is always considered
// a subset of set even if set is empty.  Subset is defined as follows: A is a subset of B if
// for all (k, v) in A, key k exists in B and key k's value in b is v.
func isSubset(set map[string]string, subset map[string]string) bool {
	if len(subset) == 0 {
		return true
	}
	if len(set) == 0 {
		// set empty, subset has items
		return false
	}

	// set and subset have items
	for k, v := range subset {
		if setValue, found := set[k]; !found || v != setValue {
			return false
		}
	}
	return true
}

// MetaSubset returns true if subset's labels and annotations are a subset of the labels and
// annotations in set.  See isSubset for the definition of subset for labels / annotations.
func MetaSubset(set meta_v1.Object, subset meta_v1.Object) bool {
	return isSubset(set.GetLabels(), subset.GetLabels()) && isSubset(set.GetAnnotations(), subset.GetAnnotations())
}

// ObjectMetaSubset returns true if the Meta field of subset is a subset of the meta field for set.
func ObjectMetaSubset(set runtime.Object, subset runtime.Object) bool {
	return MetaSubset(set.(meta_v1.Object), subset.(meta_v1.Object))
}

// IsFinalizing returns true if the object is finalizing.
func IsFinalizing(m meta_v1.Object) bool {
	return m.GetDeletionTimestamp() != nil
}
