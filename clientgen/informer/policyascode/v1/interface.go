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

// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	internalinterfaces "github.com/google/nomos/clientgen/informer/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// ClusterIAMPolicies returns a ClusterIAMPolicyInformer.
	ClusterIAMPolicies() ClusterIAMPolicyInformer
	// ClusterOrganizationPolicies returns a ClusterOrganizationPolicyInformer.
	ClusterOrganizationPolicies() ClusterOrganizationPolicyInformer
	// Folders returns a FolderInformer.
	Folders() FolderInformer
	// IAMPolicies returns a IAMPolicyInformer.
	IAMPolicies() IAMPolicyInformer
	// Organizations returns a OrganizationInformer.
	Organizations() OrganizationInformer
	// OrganizationPolicies returns a OrganizationPolicyInformer.
	OrganizationPolicies() OrganizationPolicyInformer
	// Projects returns a ProjectInformer.
	Projects() ProjectInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

// ClusterIAMPolicies returns a ClusterIAMPolicyInformer.
func (v *version) ClusterIAMPolicies() ClusterIAMPolicyInformer {
	return &clusterIAMPolicyInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// ClusterOrganizationPolicies returns a ClusterOrganizationPolicyInformer.
func (v *version) ClusterOrganizationPolicies() ClusterOrganizationPolicyInformer {
	return &clusterOrganizationPolicyInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// Folders returns a FolderInformer.
func (v *version) Folders() FolderInformer {
	return &folderInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// IAMPolicies returns a IAMPolicyInformer.
func (v *version) IAMPolicies() IAMPolicyInformer {
	return &iAMPolicyInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// Organizations returns a OrganizationInformer.
func (v *version) Organizations() OrganizationInformer {
	return &organizationInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// OrganizationPolicies returns a OrganizationPolicyInformer.
func (v *version) OrganizationPolicies() OrganizationPolicyInformer {
	return &organizationPolicyInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// Projects returns a ProjectInformer.
func (v *version) Projects() ProjectInformer {
	return &projectInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}
