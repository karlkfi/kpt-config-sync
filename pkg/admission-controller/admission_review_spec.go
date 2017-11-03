/*
Copyright 2017 The Kubernetes Authors.
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

// Admission review spec is an implementation of the admission.attributes interface
// for the admission review spec struct.
package admission_controller

import (
	"k8s.io/api/admission/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/authentication/user"
)

type AdmissionReviewSpec v1alpha1.AdmissionReviewSpec

var _ admission.Attributes = (*AdmissionReviewSpec)(nil)

func (spec *AdmissionReviewSpec) GetName() string {
	return spec.Name
}

func (spec *AdmissionReviewSpec) GetNamespace() string {
	return spec.Namespace
}

func (spec *AdmissionReviewSpec) GetResource() schema.GroupVersionResource {
	return schema.GroupVersionResource(spec.Resource)
}

func (spec *AdmissionReviewSpec) GetSubresource() string {
	return spec.SubResource
}

func (spec *AdmissionReviewSpec) GetOperation() admission.Operation {
	return admission.Operation(spec.Operation)
}

func (spec *AdmissionReviewSpec) GetObject() runtime.Object {
	return spec.Object.Object
}

func (spec *AdmissionReviewSpec) GetOldObject() runtime.Object {
	return spec.OldObject.Object
}

func (spec *AdmissionReviewSpec) GetKind() schema.GroupVersionKind {
	return schema.GroupVersionKind(spec.Kind)
}

func (spec *AdmissionReviewSpec) GetUserInfo() user.Info {

	extra := map[string][]string{}

	for key, val := range spec.UserInfo.Extra {
		extra[key] = val
	}
	return &user.DefaultInfo{
		Name:   spec.UserInfo.Username,
		UID:    spec.UserInfo.UID,
		Groups: spec.UserInfo.Groups,
		Extra:  extra,
	}
}
