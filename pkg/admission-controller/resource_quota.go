/*
Copyright 2017 The Stolos Authors.
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

// This is the Resource Quota admission controller which will do a hierarchical evaluation of quota objects
// to ensure quota is not being violated whenever resources get created/modified along the hierarchy of namespaces.
package admission_controller

import (
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	informerspolicynodev1 "github.com/google/stolos/pkg/client/informers/externalversions/policyhierarchy/v1"
	informerscorev1 "k8s.io/client-go/informers/core/v1"

	"github.com/google/stolos/pkg/resource-quota"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/admission"

	"k8s.io/kubernetes/pkg/quota"
	quotainstall "k8s.io/kubernetes/pkg/quota/install"
	"k8s.io/kubernetes/pkg/quota/generic"
	"k8s.io/kubernetes/pkg/apis/core"
)

type ResourceQuotaAdmitter struct {
	policyNodeInformer    informerspolicynodev1.PolicyNodeInformer
	resourceQuotaInformer informerscorev1.ResourceQuotaInformer
	quotaRegistry         quota.Registry
	decoder               runtime.Decoder
}

var _ Admitter = (*ResourceQuotaAdmitter)(nil)

func NewResourceQuotaAdmitter(policyNodeInformer informerspolicynodev1.PolicyNodeInformer,
	resourceQuotaInformer informerscorev1.ResourceQuotaInformer) Admitter {
	quotaConfiguration := quotainstall.NewQuotaConfigurationForAdmission()
	quotaRegistry := generic.NewRegistry(quotaConfiguration.Evaluators())

	// Decoder. Right now, only v1 and core types are needed as they are the only ones being monitored
	scheme := runtime.NewScheme()
	core_v1.AddToScheme(scheme)
	core.AddToScheme(scheme)

	decoder := serializer.NewCodecFactory(scheme).UniversalDecoder()

	return &ResourceQuotaAdmitter{
		policyNodeInformer:    policyNodeInformer,
		resourceQuotaInformer: resourceQuotaInformer,
		quotaRegistry:         quotaRegistry,
		decoder:               decoder,
	}
}

// Decides whether to admit a request
func (r *ResourceQuotaAdmitter) Admit(review admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse {
	cache, err := resource_quota.NewHierarchicalQuotaCache(r.policyNodeInformer, r.resourceQuotaInformer)
	if err != nil {
		return internalErrorDeny(err)
	}
	if review.Request == nil {
		return &admissionv1beta1.AdmissionResponse{
			Allowed: true,
		}
	}
	unpackedSpec := unpackRawSpec(r.decoder, *review.Request)
	reviewSpec := AdmissionReviewSpec(unpackedSpec)
	attributes := admission.Attributes(&reviewSpec)
	evaluator := r.quotaRegistry.Get(attributes.GetResource().GroupResource())
	if evaluator != nil && evaluator.Handles(attributes) {
		newUsage, err := evaluator.Usage(attributes.GetObject())
		if err != nil {
			return internalErrorDeny(err)
		}

		v1NewUsage := core_v1.ResourceList{}
		for key, val := range newUsage {
			v1NewUsage[core_v1.ResourceName(key)] = val
		}

		admitError := cache.Admit(review.Request.Namespace, v1NewUsage)

		if admitError != nil {
			return &admissionv1beta1.AdmissionResponse{
				Allowed: false,
				Result: &metav1.Status{
					Message: admitError.Error(),
					Reason:  metav1.StatusReason(metav1.StatusReasonForbidden),
				},
			}
		}
	}
	return &admissionv1beta1.AdmissionResponse{
		Allowed: true,
	}
}

func internalErrorDeny(err error) *admissionv1beta1.AdmissionResponse {
	return &admissionv1beta1.AdmissionResponse{
		Allowed: false,
		Result: &metav1.Status{
			Message: err.Error(),
			Reason:  metav1.StatusReason(metav1.StatusReasonInternalError),
		},
	}
}

