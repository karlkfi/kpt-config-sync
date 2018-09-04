// +build !ignore_autogenerated

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

// This file was autogenerated by deepcopy-gen. Do not edit it manually!

package v1

import (
	core_v1 "k8s.io/api/core/v1"
	v1beta1 "k8s.io/api/extensions/v1beta1"
	rbac_v1 "k8s.io/api/rbac/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AllPolicies) DeepCopyInto(out *AllPolicies) {
	*out = *in
	if in.PolicyNodes != nil {
		in, out := &in.PolicyNodes, &out.PolicyNodes
		*out = make(map[string]PolicyNode, len(*in))
		for key, val := range *in {
			newVal := new(PolicyNode)
			val.DeepCopyInto(newVal)
			(*out)[key] = *newVal
		}
	}
	if in.ClusterPolicy != nil {
		in, out := &in.ClusterPolicy, &out.ClusterPolicy
		if *in == nil {
			*out = nil
		} else {
			*out = new(ClusterPolicy)
			(*in).DeepCopyInto(*out)
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AllPolicies.
func (in *AllPolicies) DeepCopy() *AllPolicies {
	if in == nil {
		return nil
	}
	out := new(AllPolicies)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterPolicy) DeepCopyInto(out *ClusterPolicy) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterPolicy.
func (in *ClusterPolicy) DeepCopy() *ClusterPolicy {
	if in == nil {
		return nil
	}
	out := new(ClusterPolicy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterPolicy) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterPolicyList) DeepCopyInto(out *ClusterPolicyList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ClusterPolicy, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterPolicyList.
func (in *ClusterPolicyList) DeepCopy() *ClusterPolicyList {
	if in == nil {
		return nil
	}
	out := new(ClusterPolicyList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterPolicyList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterPolicySpec) DeepCopyInto(out *ClusterPolicySpec) {
	*out = *in
	if in.ClusterRolesV1 != nil {
		in, out := &in.ClusterRolesV1, &out.ClusterRolesV1
		*out = make([]rbac_v1.ClusterRole, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.ClusterRoleBindingsV1 != nil {
		in, out := &in.ClusterRoleBindingsV1, &out.ClusterRoleBindingsV1
		*out = make([]rbac_v1.ClusterRoleBinding, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.PodSecurityPoliciesV1Beta1 != nil {
		in, out := &in.PodSecurityPoliciesV1Beta1, &out.PodSecurityPoliciesV1Beta1
		*out = make([]v1beta1.PodSecurityPolicy, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.ImportTime.DeepCopyInto(&out.ImportTime)
	if in.Resources != nil {
		in, out := &in.Resources, &out.Resources
		*out = make([]GenericResources, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterPolicySpec.
func (in *ClusterPolicySpec) DeepCopy() *ClusterPolicySpec {
	if in == nil {
		return nil
	}
	out := new(ClusterPolicySpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterPolicyStatus) DeepCopyInto(out *ClusterPolicyStatus) {
	*out = *in
	if in.SyncErrors != nil {
		in, out := &in.SyncErrors, &out.SyncErrors
		*out = make([]ClusterPolicySyncError, len(*in))
		copy(*out, *in)
	}
	in.SyncTime.DeepCopyInto(&out.SyncTime)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterPolicyStatus.
func (in *ClusterPolicyStatus) DeepCopy() *ClusterPolicyStatus {
	if in == nil {
		return nil
	}
	out := new(ClusterPolicyStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterPolicySyncError) DeepCopyInto(out *ClusterPolicySyncError) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterPolicySyncError.
func (in *ClusterPolicySyncError) DeepCopy() *ClusterPolicySyncError {
	if in == nil {
		return nil
	}
	out := new(ClusterPolicySyncError)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GenericResources) DeepCopyInto(out *GenericResources) {
	*out = *in
	if in.Versions != nil {
		in, out := &in.Versions, &out.Versions
		*out = make([]GenericVersionResources, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GenericResources.
func (in *GenericResources) DeepCopy() *GenericResources {
	if in == nil {
		return nil
	}
	out := new(GenericResources)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GenericVersionResources) DeepCopyInto(out *GenericVersionResources) {
	*out = *in
	if in.Objects != nil {
		in, out := &in.Objects, &out.Objects
		*out = make([]runtime.RawExtension, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GenericVersionResources.
func (in *GenericVersionResources) DeepCopy() *GenericVersionResources {
	if in == nil {
		return nil
	}
	out := new(GenericVersionResources)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NamespaceSelector) DeepCopyInto(out *NamespaceSelector) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NamespaceSelector.
func (in *NamespaceSelector) DeepCopy() *NamespaceSelector {
	if in == nil {
		return nil
	}
	out := new(NamespaceSelector)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NamespaceSelector) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NamespaceSelectorList) DeepCopyInto(out *NamespaceSelectorList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]NamespaceSelector, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NamespaceSelectorList.
func (in *NamespaceSelectorList) DeepCopy() *NamespaceSelectorList {
	if in == nil {
		return nil
	}
	out := new(NamespaceSelectorList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NamespaceSelectorList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NamespaceSelectorSpec) DeepCopyInto(out *NamespaceSelectorSpec) {
	*out = *in
	in.Selector.DeepCopyInto(&out.Selector)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NamespaceSelectorSpec.
func (in *NamespaceSelectorSpec) DeepCopy() *NamespaceSelectorSpec {
	if in == nil {
		return nil
	}
	out := new(NamespaceSelectorSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NomosConfig) DeepCopyInto(out *NomosConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NomosConfig.
func (in *NomosConfig) DeepCopy() *NomosConfig {
	if in == nil {
		return nil
	}
	out := new(NomosConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NomosConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NomosConfigSpec) DeepCopyInto(out *NomosConfigSpec) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NomosConfigSpec.
func (in *NomosConfigSpec) DeepCopy() *NomosConfigSpec {
	if in == nil {
		return nil
	}
	out := new(NomosConfigSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PolicyNode) DeepCopyInto(out *PolicyNode) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PolicyNode.
func (in *PolicyNode) DeepCopy() *PolicyNode {
	if in == nil {
		return nil
	}
	out := new(PolicyNode)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PolicyNode) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PolicyNodeList) DeepCopyInto(out *PolicyNodeList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]PolicyNode, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PolicyNodeList.
func (in *PolicyNodeList) DeepCopy() *PolicyNodeList {
	if in == nil {
		return nil
	}
	out := new(PolicyNodeList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PolicyNodeList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PolicyNodeSpec) DeepCopyInto(out *PolicyNodeSpec) {
	*out = *in
	if in.RolesV1 != nil {
		in, out := &in.RolesV1, &out.RolesV1
		*out = make([]rbac_v1.Role, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.RoleBindingsV1 != nil {
		in, out := &in.RoleBindingsV1, &out.RoleBindingsV1
		*out = make([]rbac_v1.RoleBinding, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.ResourceQuotaV1 != nil {
		in, out := &in.ResourceQuotaV1, &out.ResourceQuotaV1
		if *in == nil {
			*out = nil
		} else {
			*out = new(core_v1.ResourceQuota)
			(*in).DeepCopyInto(*out)
		}
	}
	in.ImportTime.DeepCopyInto(&out.ImportTime)
	if in.Resources != nil {
		in, out := &in.Resources, &out.Resources
		*out = make([]GenericResources, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PolicyNodeSpec.
func (in *PolicyNodeSpec) DeepCopy() *PolicyNodeSpec {
	if in == nil {
		return nil
	}
	out := new(PolicyNodeSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PolicyNodeStatus) DeepCopyInto(out *PolicyNodeStatus) {
	*out = *in
	if in.SyncTokens != nil {
		in, out := &in.SyncTokens, &out.SyncTokens
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.SyncErrors != nil {
		in, out := &in.SyncErrors, &out.SyncErrors
		*out = make([]PolicyNodeSyncError, len(*in))
		copy(*out, *in)
	}
	in.SyncTime.DeepCopyInto(&out.SyncTime)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PolicyNodeStatus.
func (in *PolicyNodeStatus) DeepCopy() *PolicyNodeStatus {
	if in == nil {
		return nil
	}
	out := new(PolicyNodeStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PolicyNodeSyncError) DeepCopyInto(out *PolicyNodeSyncError) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PolicyNodeSyncError.
func (in *PolicyNodeSyncError) DeepCopy() *PolicyNodeSyncError {
	if in == nil {
		return nil
	}
	out := new(PolicyNodeSyncError)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Sync) DeepCopyInto(out *Sync) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Sync.
func (in *Sync) DeepCopy() *Sync {
	if in == nil {
		return nil
	}
	out := new(Sync)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Sync) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SyncGroup) DeepCopyInto(out *SyncGroup) {
	*out = *in
	in.Kinds.DeepCopyInto(&out.Kinds)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SyncGroup.
func (in *SyncGroup) DeepCopy() *SyncGroup {
	if in == nil {
		return nil
	}
	out := new(SyncGroup)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SyncGroupKindStatus) DeepCopyInto(out *SyncGroupKindStatus) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SyncGroupKindStatus.
func (in *SyncGroupKindStatus) DeepCopy() *SyncGroupKindStatus {
	if in == nil {
		return nil
	}
	out := new(SyncGroupKindStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SyncKind) DeepCopyInto(out *SyncKind) {
	*out = *in
	if in.Versions != nil {
		in, out := &in.Versions, &out.Versions
		*out = make([]SyncVersion, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SyncKind.
func (in *SyncKind) DeepCopy() *SyncKind {
	if in == nil {
		return nil
	}
	out := new(SyncKind)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SyncList) DeepCopyInto(out *SyncList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Sync, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SyncList.
func (in *SyncList) DeepCopy() *SyncList {
	if in == nil {
		return nil
	}
	out := new(SyncList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *SyncList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SyncSpec) DeepCopyInto(out *SyncSpec) {
	*out = *in
	if in.Groups != nil {
		in, out := &in.Groups, &out.Groups
		*out = make([]SyncGroup, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SyncSpec.
func (in *SyncSpec) DeepCopy() *SyncSpec {
	if in == nil {
		return nil
	}
	out := new(SyncSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SyncStatus) DeepCopyInto(out *SyncStatus) {
	*out = *in
	if in.GroupKinds != nil {
		in, out := &in.GroupKinds, &out.GroupKinds
		*out = make([]SyncGroupKindStatus, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SyncStatus.
func (in *SyncStatus) DeepCopy() *SyncStatus {
	if in == nil {
		return nil
	}
	out := new(SyncStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SyncVersion) DeepCopyInto(out *SyncVersion) {
	*out = *in
	if in.CompareFields != nil {
		in, out := &in.CompareFields, &out.CompareFields
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SyncVersion.
func (in *SyncVersion) DeepCopy() *SyncVersion {
	if in == nil {
		return nil
	}
	out := new(SyncVersion)
	in.DeepCopyInto(out)
	return out
}
