//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
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
func (in *ContainerResourcesSpec) DeepCopyInto(out *ContainerResourcesSpec) {
	*out = *in
	out.CPURequest = in.CPURequest.DeepCopy()
	out.MemoryRequest = in.MemoryRequest.DeepCopy()
	out.CPULimit = in.CPULimit.DeepCopy()
	out.MemoryLimit = in.MemoryLimit.DeepCopy()
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ContainerResourcesSpec.
func (in *ContainerResourcesSpec) DeepCopy() *ContainerResourcesSpec {
	if in == nil {
		return nil
	}
	out := new(ContainerResourcesSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ErrorSummary) DeepCopyInto(out *ErrorSummary) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ErrorSummary.
func (in *ErrorSummary) DeepCopy() *ErrorSummary {
	if in == nil {
		return nil
	}
	out := new(ErrorSummary)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Git) DeepCopyInto(out *Git) {
	*out = *in
	out.Period = in.Period
	out.SecretRef = in.SecretRef
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Git.
func (in *Git) DeepCopy() *Git {
	if in == nil {
		return nil
	}
	out := new(Git)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GitSourceStatus) DeepCopyInto(out *GitSourceStatus) {
	*out = *in
	out.Git = in.Git
	in.LastUpdate.DeepCopyInto(&out.LastUpdate)
	if in.Errors != nil {
		in, out := &in.Errors, &out.Errors
		*out = make([]ConfigSyncError, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.ErrorSummary != nil {
		in, out := &in.ErrorSummary, &out.ErrorSummary
		*out = new(ErrorSummary)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GitSourceStatus.
func (in *GitSourceStatus) DeepCopy() *GitSourceStatus {
	if in == nil {
		return nil
	}
	out := new(GitSourceStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GitStatus) DeepCopyInto(out *GitStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GitStatus.
func (in *GitStatus) DeepCopy() *GitStatus {
	if in == nil {
		return nil
	}
	out := new(GitStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GitSyncStatus) DeepCopyInto(out *GitSyncStatus) {
	*out = *in
	out.Git = in.Git
	in.LastUpdate.DeepCopyInto(&out.LastUpdate)
	if in.Errors != nil {
		in, out := &in.Errors, &out.Errors
		*out = make([]ConfigSyncError, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.ErrorSummary != nil {
		in, out := &in.ErrorSummary, &out.ErrorSummary
		*out = new(ErrorSummary)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GitSyncStatus.
func (in *GitSyncStatus) DeepCopy() *GitSyncStatus {
	if in == nil {
		return nil
	}
	out := new(GitSyncStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OverrideSpec) DeepCopyInto(out *OverrideSpec) {
	*out = *in
	if in.Resources != nil {
		in, out := &in.Resources, &out.Resources
		*out = make([]ContainerResourcesSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.GitSyncDepth != nil {
		in, out := &in.GitSyncDepth, &out.GitSyncDepth
		*out = new(int64)
		**out = **in
	}
	out.ReconcileTimeout = in.ReconcileTimeout
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OverrideSpec.
func (in *OverrideSpec) DeepCopy() *OverrideSpec {
	if in == nil {
		return nil
	}
	out := new(OverrideSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RenderingStatus) DeepCopyInto(out *RenderingStatus) {
	*out = *in
	out.Git = in.Git
	in.LastUpdate.DeepCopyInto(&out.LastUpdate)
	if in.Errors != nil {
		in, out := &in.Errors, &out.Errors
		*out = make([]ConfigSyncError, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.ErrorSummary != nil {
		in, out := &in.ErrorSummary, &out.ErrorSummary
		*out = new(ErrorSummary)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RenderingStatus.
func (in *RenderingStatus) DeepCopy() *RenderingStatus {
	if in == nil {
		return nil
	}
	out := new(RenderingStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RepoSync) DeepCopyInto(out *RepoSync) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
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
	if in.Errors != nil {
		in, out := &in.Errors, &out.Errors
		*out = make([]ConfigSyncError, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.ErrorSourceRefs != nil {
		in, out := &in.ErrorSourceRefs, &out.ErrorSourceRefs
		*out = make([]ErrorSource, len(*in))
		copy(*out, *in)
	}
	if in.ErrorSummary != nil {
		in, out := &in.ErrorSummary, &out.ErrorSummary
		*out = new(ErrorSummary)
		**out = **in
	}
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
func (in *RepoSyncSpec) DeepCopyInto(out *RepoSyncSpec) {
	*out = *in
	in.SyncSpec.DeepCopyInto(&out.SyncSpec)
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
func (in *RepoSyncStatus) DeepCopyInto(out *RepoSyncStatus) {
	*out = *in
	in.SyncStatus.DeepCopyInto(&out.SyncStatus)
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]RepoSyncCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RepoSyncStatus.
func (in *RepoSyncStatus) DeepCopy() *RepoSyncStatus {
	if in == nil {
		return nil
	}
	out := new(RepoSyncStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceRef) DeepCopyInto(out *ResourceRef) {
	*out = *in
	out.GVK = in.GVK
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceRef.
func (in *ResourceRef) DeepCopy() *ResourceRef {
	if in == nil {
		return nil
	}
	out := new(ResourceRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RootSync) DeepCopyInto(out *RootSync) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
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
func (in *RootSyncCondition) DeepCopyInto(out *RootSyncCondition) {
	*out = *in
	in.LastUpdateTime.DeepCopyInto(&out.LastUpdateTime)
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
	if in.Errors != nil {
		in, out := &in.Errors, &out.Errors
		*out = make([]ConfigSyncError, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.ErrorSourceRefs != nil {
		in, out := &in.ErrorSourceRefs, &out.ErrorSourceRefs
		*out = make([]ErrorSource, len(*in))
		copy(*out, *in)
	}
	if in.ErrorSummary != nil {
		in, out := &in.ErrorSummary, &out.ErrorSummary
		*out = new(ErrorSummary)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RootSyncCondition.
func (in *RootSyncCondition) DeepCopy() *RootSyncCondition {
	if in == nil {
		return nil
	}
	out := new(RootSyncCondition)
	in.DeepCopyInto(out)
	return out
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
	in.SyncSpec.DeepCopyInto(&out.SyncSpec)
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
	in.SyncStatus.DeepCopyInto(&out.SyncStatus)
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]RootSyncCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
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

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SecretReference) DeepCopyInto(out *SecretReference) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SecretReference.
func (in *SecretReference) DeepCopy() *SecretReference {
	if in == nil {
		return nil
	}
	out := new(SecretReference)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SyncSpec) DeepCopyInto(out *SyncSpec) {
	*out = *in
	out.Git = in.Git
	in.Override.DeepCopyInto(&out.Override)
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
	in.Source.DeepCopyInto(&out.Source)
	in.Rendering.DeepCopyInto(&out.Rendering)
	in.Sync.DeepCopyInto(&out.Sync)
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
