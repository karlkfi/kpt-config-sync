/*
Copyright 2018 The Kubernetes Authors.

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

package v1alpha1

import (
	"github.com/google/nomos/pkg/installer/cluster-operators/pkg/operators"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GitConfig contains the configs needed by GitPolicyImporter.
//
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
type GitConfig struct {
	// SyncRepo is the git repository URL to sync from. Required.

	// +kubebuilder:validation:Pattern=^(((https?|git|ssh):\/\/)|git@)
	SyncRepo string `json:"syncRepo"`

	// SyncBranch is the branch to sync from.  Default: "master".
	SyncBranch string `json:"syncBranch,omitempty"`

	// PolicyDir is the absolute path of the directory that contains
	// the local policy.  Default: the root directory of the repo.
	PolicyDir string `json:"policyDir,omitempty"`

	// SyncWaitSeconds is the time duration in seconds between consecutive
	// syncs.  Default: 15 seconds.
	// Note that SyncWaitSecs is not a time.Duration on purpose. This provides
	// a reminder to developers that customers specify this value using
	// using integers like "3" in their Nomos YAML. However, time.Duration
	// is at a nanosecond granularity, and it's easy to introduce a bug where
	// it looks like the code is dealing with seconds but its actually nanoseconds
	// (or vice versa).
	SyncWaitSecs int `json:"syncWait,omitempty"`

	// SyncRev is the git revision (tag or hash) to check out. Default: HEAD.
	SyncRev string `json:"syncRev,omitempty"`

	// SecretType is the type of secret configured for access to the Git repo.
	// Must be one of ssh, password, or cookiefile. Required.
	// The validation of this is case-sensitive.

	// +kubebuilder:validation:Pattern=^(ssh|password|cookiefile)$
	SecretType string `json:"secretType"`
}

// NomosSpec defines the desired state of Nomos.
type NomosSpec struct {
	// Important: Run "make" to regenerate code after modifying this file
	operators.CommonSpec

	// The user account that will drive the installation.  Required to insert
	// cluster administration role bindings into GKE clusters.
	User string `json:"user"`

	// Git contains configuration specific to importing policies from a Git repo.
	Git GitConfig `json:"git,omitempty"`
}

// NomosStatus defines the observed state of Nomos.
type NomosStatus struct {
	// Important: Run "make" to regenerate code after modifying this file
	operators.CommonStatus
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Nomos is the Schema for the Nomos API.
// +k8s:openapi-gen=true
type Nomos struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   NomosSpec   `json:"spec"`
	Status NomosStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NomosList contains a list of Nomos.
type NomosList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Nomos `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Nomos{}, &NomosList{})
}

// CommonSpec ...
func (c *Nomos) CommonSpec() operators.CommonSpec {
	return c.Spec.CommonSpec
}

// GetCommonStatus ...
func (c *Nomos) GetCommonStatus() operators.CommonStatus {
	return c.Status.CommonStatus
}

// SetCommonStatus ...
func (c *Nomos) SetCommonStatus(s operators.CommonStatus) {
	c.Status.CommonStatus = s
}

// Empty ...
func (g *GitConfig) Empty() bool {
	return g.SyncRepo == "" && g.PolicyDir == ""
}
