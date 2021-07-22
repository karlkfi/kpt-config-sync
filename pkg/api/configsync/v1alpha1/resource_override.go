package v1alpha1

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
	// Must be "reconciler" or "git-sync".
	//
	// +kubebuilder:validation:Pattern=^(reconciler|git-sync|hydration-controller)$
	// +optional
	ContainerName string `json:"containerName,omitempty"`
	// cpuLimit allows one to override the CPU limit of a container
	// +optional
	CPULimit resource.Quantity `json:"cpuLimit,omitempty"`
	// memoryLimit allows one to override the memory limit of a container
	// +optional
	MemoryLimit resource.Quantity `json:"memoryLimit,omitempty"`
}
