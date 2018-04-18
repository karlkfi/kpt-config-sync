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
func (in *ClusterPolicies) DeepCopyInto(out *ClusterPolicies) {
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
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterPolicies.
func (in *ClusterPolicies) DeepCopy() *ClusterPolicies {
	if in == nil {
		return nil
	}
	out := new(ClusterPolicies)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterPolicy) DeepCopyInto(out *ClusterPolicy) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
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
	in.Policies.DeepCopyInto(&out.Policies)
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
func (in *Policies) DeepCopyInto(out *Policies) {
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
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Policies.
func (in *Policies) DeepCopy() *Policies {
	if in == nil {
		return nil
	}
	out := new(Policies)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PolicyNode) DeepCopyInto(out *PolicyNode) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
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
	in.Policies.DeepCopyInto(&out.Policies)
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
