/*
Copyright 2017 The Nomos Authors.
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
package admissioncontroller

import (
	"github.com/golang/glog"
	"k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/authentication/user"
)

type AdmissionReviewSpec v1beta1.AdmissionRequest

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

// getAttributes returns admissions attributes initialized from the given request and decoder.
func getAttributes(decoder runtime.Decoder, request v1beta1.AdmissionRequest) admission.Attributes {
	unpackedSpec := unpackRawSpec(decoder, request)
	reviewSpec := AdmissionReviewSpec(unpackedSpec)
	return admission.Attributes(&reviewSpec)
}

// For convenience, unpack the object and oldObject raw bytes into a runtimeobject (so that this doesn't
// need to happen every time we call GetObject() above.) This is a utility method that should be performed
// before casting the spec to AdmissionReviewSpec.
//
// This unpack may currently not work in many cases due to a bug in Kubernetes that passes in the wrong version of
// the object (internal) in the raw extension.
func unpackRawSpec(decoder runtime.Decoder, request v1beta1.AdmissionRequest) v1beta1.AdmissionRequest {
	if request.Object.Object != nil {
		// Already unpacked
		return request
	}
	if len(request.Object.Raw) > 0 {
		request.Object.Object = unpackRawBytes(decoder, schema.GroupVersionKind(request.Kind), request.Object.Raw)
	}

	if len(request.OldObject.Raw) > 0 {
		request.OldObject.Object = unpackRawBytes(decoder, schema.GroupVersionKind(request.Kind), request.OldObject.Raw)
	}

	return request
}

// Helper method for unpackRawSpec
func unpackRawBytes(decoder runtime.Decoder, gvk schema.GroupVersionKind, raw []byte) runtime.Object {
	obj, _, err := decoder.Decode(raw, &gvk, nil)
	if err != nil {
		glog.V(7).Infof("Error un-marshalling review object, continuing: %s", err)
		return nil
	}

	return obj
}
