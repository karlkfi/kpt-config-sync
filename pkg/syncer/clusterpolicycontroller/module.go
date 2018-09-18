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
// Reviewed by sunilarora

package clusterpolicycontroller

import (
	policyhierarchyv1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/syncer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Module is a type specific implementation for cluster synchronization.
// Each module is responsible for synchronizing only one resource type.
type Module interface {
	syncer.Module

	// Extract returns the instances of the type that this module is going
	// to be synchronizing. Since it operates on API types, they should all
	// satisfy the metav1.Object interface.
	Extract(*policyhierarchyv1.ClusterPolicy) []metav1.Object
}
