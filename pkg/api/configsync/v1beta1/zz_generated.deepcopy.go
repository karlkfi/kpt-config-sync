//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Code generated by controller-gen. DO NOT EDIT.

package v1beta1

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1"
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
func (in *Helm) DeepCopyInto(out *Helm) {
	*out = *in
	in.Values.DeepCopyInto(&out.Values)
	if in.ValuesFiles != nil {
		in, out := &in.ValuesFiles, &out.ValuesFiles
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	out.Period = in.Period
	out.SecretRef = in.SecretRef
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Helm.
func (in *Helm) DeepCopy() *Helm {
	if in == nil {
		return nil
	}
	out := new(Helm)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HelmStatus) DeepCopyInto(out *HelmStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HelmStatus.
func (in *HelmStatus) DeepCopy() *HelmStatus {
	if in == nil {
		return nil
	}
	out := new(HelmStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Oci) DeepCopyInto(out *Oci) {
	*out = *in
	out.Period = in.Period
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Oci.
func (in *Oci) DeepCopy() *Oci {
	if in == nil {
		return nil
	}
	out := new(Oci)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OciStatus) DeepCopyInto(out *OciStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OciStatus.
func (in *OciStatus) DeepCopy() *OciStatus {
	if in == nil {
		return nil
	}
	out := new(OciStatus)
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
	if in.ReconcileTimeout != nil {
		in, out := &in.ReconcileTimeout, &out.ReconcileTimeout
		*out = new(v1.Duration)
		**out = **in
	}
	if in.EnableShellInRendering != nil {
		in, out := &in.EnableShellInRendering, &out.EnableShellInRendering
		*out = new(bool)
		**out = **in
	}
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
	if in.Git != nil {
		in, out := &in.Git, &out.Git
		*out = new(GitStatus)
		**out = **in
	}
	if in.Oci != nil {
		in, out := &in.Oci, &out.Oci
		*out = new(OciStatus)
		**out = **in
	}
	if in.Helm != nil {
		in, out := &in.Helm, &out.Helm
		*out = new(HelmStatus)
		**out = **in
	}
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
	if in.Git != nil {
		in, out := &in.Git, &out.Git
		*out = new(Git)
		**out = **in
	}
	if in.Oci != nil {
		in, out := &in.Oci, &out.Oci
		*out = new(Oci)
		**out = **in
	}
	if in.Helm != nil {
		in, out := &in.Helm, &out.Helm
		*out = new(Helm)
		(*in).DeepCopyInto(*out)
	}
	in.Override.DeepCopyInto(&out.Override)
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
	in.Status.DeepCopyInto(&out.Status)
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
	if in.Git != nil {
		in, out := &in.Git, &out.Git
		*out = new(Git)
		**out = **in
	}
	if in.Oci != nil {
		in, out := &in.Oci, &out.Oci
		*out = new(Oci)
		**out = **in
	}
	if in.Helm != nil {
		in, out := &in.Helm, &out.Helm
		*out = new(Helm)
		(*in).DeepCopyInto(*out)
	}
	in.Override.DeepCopyInto(&out.Override)
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
	in.Status.DeepCopyInto(&out.Status)
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
func (in *SourceStatus) DeepCopyInto(out *SourceStatus) {
	*out = *in
	if in.Git != nil {
		in, out := &in.Git, &out.Git
		*out = new(GitStatus)
		**out = **in
	}
	if in.Oci != nil {
		in, out := &in.Oci, &out.Oci
		*out = new(OciStatus)
		**out = **in
	}
	if in.Helm != nil {
		in, out := &in.Helm, &out.Helm
		*out = new(HelmStatus)
		**out = **in
	}
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

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SourceStatus.
func (in *SourceStatus) DeepCopy() *SourceStatus {
	if in == nil {
		return nil
	}
	out := new(SourceStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Status) DeepCopyInto(out *Status) {
	*out = *in
	in.Source.DeepCopyInto(&out.Source)
	in.Rendering.DeepCopyInto(&out.Rendering)
	in.Sync.DeepCopyInto(&out.Sync)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Status.
func (in *Status) DeepCopy() *Status {
	if in == nil {
		return nil
	}
	out := new(Status)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SyncStatus) DeepCopyInto(out *SyncStatus) {
	*out = *in
	if in.Git != nil {
		in, out := &in.Git, &out.Git
		*out = new(GitStatus)
		**out = **in
	}
	if in.Oci != nil {
		in, out := &in.Oci, &out.Oci
		*out = new(OciStatus)
		**out = **in
	}
	if in.Helm != nil {
		in, out := &in.Helm, &out.Helm
		*out = new(HelmStatus)
		**out = **in
	}
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

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SyncStatus.
func (in *SyncStatus) DeepCopy() *SyncStatus {
	if in == nil {
		return nil
	}
	out := new(SyncStatus)
	in.DeepCopyInto(out)
	return out
}
