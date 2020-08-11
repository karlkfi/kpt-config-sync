// +build !ignore_autogenerated

// Code generated by controller-gen. DO NOT EDIT.

package v1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConfigSyncError) DeepCopyInto(out *ConfigSyncError) {
	*out = *in
	if in.Resources != nil {
		in, out := &in.Resources, &out.Resources
		*out = make([]ResourceRef, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConfigSyncError.
func (in *ConfigSyncError) DeepCopy() *ConfigSyncError {
	if in == nil {
		return nil
	}
	out := new(ConfigSyncError)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RepoSync) DeepCopyInto(out *RepoSync) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RepoSync.
func (in *RepoSync) DeepCopy() *RepoSync {
	if in == nil {
		return nil
	}
	out := new(RepoSync)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RepoSync) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RepoSyncCondition) DeepCopyInto(out *RepoSyncCondition) {
	*out = *in
	in.LastUpdateTime.DeepCopyInto(&out.LastUpdateTime)
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RepoSyncCondition.
func (in *RepoSyncCondition) DeepCopy() *RepoSyncCondition {
	if in == nil {
		return nil
	}
	out := new(RepoSyncCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RepoSyncList) DeepCopyInto(out *RepoSyncList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]RepoSync, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RepoSyncList.
func (in *RepoSyncList) DeepCopy() *RepoSyncList {
	if in == nil {
		return nil
	}
	out := new(RepoSyncList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RepoSyncList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RepoSyncSourceStatus) DeepCopyInto(out *RepoSyncSourceStatus) {
	*out = *in
	out.Git = in.Git
	if in.Errors != nil {
		in, out := &in.Errors, &out.Errors
		*out = make([]ConfigSyncError, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RepoSyncSourceStatus.
func (in *RepoSyncSourceStatus) DeepCopy() *RepoSyncSourceStatus {
	if in == nil {
		return nil
	}
	out := new(RepoSyncSourceStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RepoSyncSpec) DeepCopyInto(out *RepoSyncSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RepoSyncSpec.
func (in *RepoSyncSpec) DeepCopy() *RepoSyncSpec {
	if in == nil {
		return nil
	}
	out := new(RepoSyncSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RepoSyncSyncStatus) DeepCopyInto(out *RepoSyncSyncStatus) {
	*out = *in
	in.LastUpdate.DeepCopyInto(&out.LastUpdate)
	if in.Errors != nil {
		in, out := &in.Errors, &out.Errors
		*out = make([]ConfigSyncError, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RepoSyncSyncStatus.
func (in *RepoSyncSyncStatus) DeepCopy() *RepoSyncSyncStatus {
	if in == nil {
		return nil
	}
	out := new(RepoSyncSyncStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RepoSyncsStatus) DeepCopyInto(out *RepoSyncsStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]RepoSyncCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.Source.DeepCopyInto(&out.Source)
	in.Sync.DeepCopyInto(&out.Sync)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RepoSyncsStatus.
func (in *RepoSyncsStatus) DeepCopy() *RepoSyncsStatus {
	if in == nil {
		return nil
	}
	out := new(RepoSyncsStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RootSync) DeepCopyInto(out *RootSync) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RootSync.
func (in *RootSync) DeepCopy() *RootSync {
	if in == nil {
		return nil
	}
	out := new(RootSync)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RootSync) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RootSyncList) DeepCopyInto(out *RootSyncList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]RootSync, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RootSyncList.
func (in *RootSyncList) DeepCopy() *RootSyncList {
	if in == nil {
		return nil
	}
	out := new(RootSyncList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RootSyncList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RootSyncSpec) DeepCopyInto(out *RootSyncSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RootSyncSpec.
func (in *RootSyncSpec) DeepCopy() *RootSyncSpec {
	if in == nil {
		return nil
	}
	out := new(RootSyncSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RootSyncStatus) DeepCopyInto(out *RootSyncStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RootSyncStatus.
func (in *RootSyncStatus) DeepCopy() *RootSyncStatus {
	if in == nil {
		return nil
	}
	out := new(RootSyncStatus)
	in.DeepCopyInto(out)
	return out
}
