// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1beta1

import "k8s.io/apimachinery/pkg/api/resource"

// OverrideSpec allows to override the settings for a reconciler pod
type OverrideSpec struct {
	// resources allow one to override the resource requirements for the containers in a reconciler pod.
	// +optional
	Resources []ContainerResourcesSpec `json:"resources,omitempty"`

	// gitSyncDepth allows one to override the number of git commits to fetch.
	// Must be no less than 0.
	// Config Sync would do a full clone if this field is 0, and a shallow
	// clone if this field is greater than 0.
	// If this field is not provided, Config Sync would configure it automatically.
	//
	// +kubebuilder:validation:Minimum=0
	// +optional
	GitSyncDepth *int64 `json:"gitSyncDepth,omitempty"`
}

// ContainerResourcesSpec allows to override the resource requirements for a container
type ContainerResourcesSpec struct {
	// containerName specifies the name of a container whose resource requirements will be overridden.
	// Must be "reconciler", "git-sync", or "hydration-controller".
	//
	// +kubebuilder:validation:Pattern=^(reconciler|git-sync|hydration-controller)$
	// +optional
	ContainerName string `json:"containerName,omitempty"`
	// cpuRequest allows one to override the CPU request of a container
	// +optional
	CPURequest resource.Quantity `json:"cpuRequest,omitempty"`
	// memoryRequest allows one to override the memory request of a container
	// +optional
	MemoryRequest resource.Quantity `json:"memoryRequest,omitempty"`
	// cpuLimit allows one to override the CPU limit of a container
	// +optional
	CPULimit resource.Quantity `json:"cpuLimit,omitempty"`
	// memoryLimit allows one to override the memory limit of a container
	// +optional
	MemoryLimit resource.Quantity `json:"memoryLimit,omitempty"`
}
