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

// This is the Resource Quota admission controller which will do a hierarchical evaluation of quota objects
// to ensure quota is not being violated whenever resources get created/modified along the hierarchy of namespaces.
package admission_controller

import (
	admissionv1alpha1 "k8s.io/api/admission/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	informerspolicynodev1 "github.com/google/stolos/pkg/client/informers/externalversions/k8us/v1"
	informerscorev1 "k8s.io/client-go/informers/core/v1"

	"github.com/golang/glog"
	"github.com/google/stolos/pkg/resource-quota"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/kubernetes/pkg/quota"
	quotainstall "k8s.io/kubernetes/pkg/quota/install"
)

type ResourceQuotaAdmitter struct {
	policyNodeInformer    informerspolicynodev1.PolicyNodeInformer
	resourceQuotaInformer informerscorev1.ResourceQuotaInformer
	quotaRegistry         quota.Registry
}

var _ Admitter = (*ResourceQuotaAdmitter)(nil)

func NewResourceQuotaAdmitter(policyNodeInformer informerspolicynodev1.PolicyNodeInformer,
	resourceQuotaInformer informerscorev1.ResourceQuotaInformer) Admitter {
	// Nil, because we don't need to do any watches, we will only be doing evaluation checks.
	quotaRegistry := quotainstall.NewRegistry(nil, nil)
	return &ResourceQuotaAdmitter{
		policyNodeInformer:    policyNodeInformer,
		resourceQuotaInformer: resourceQuotaInformer,
		quotaRegistry:         quotaRegistry,
	}
}

// Decides whether to admit a request
func (r *ResourceQuotaAdmitter) Admit(review admissionv1alpha1.AdmissionReview) *admissionv1alpha1.AdmissionReviewStatus {
	cache, err := resource_quota.NewHierarchicalQuotaCache(r.policyNodeInformer, r.resourceQuotaInformer)
	if err != nil {
		return internalErrorDeny(err)
	}

	reviewSpec := AdmissionReviewSpec(review.Spec)
	attributes := admission.Attributes(&reviewSpec)
	evaluator := r.quotaRegistry.Evaluators()[attributes.GetKind().GroupKind()]

	if evaluator != nil && evaluator.Handles(attributes) {
		newUsage, err := evaluator.Usage(attributes.GetObject())
		if err != nil {
			// Until b/68666077 is done, this can happen for legit objects. So not throwing error right now.
			// return internalErrorDeny(err)
			glog.Infof("Got error calculating usage due to %s but ignoring", err)
			return &admissionv1alpha1.AdmissionReviewStatus{
				Allowed: true,
			}
		}

		v1NewUsage := core_v1.ResourceList{}
		for key, val := range newUsage {
			v1NewUsage[core_v1.ResourceName(key)] = val
		}

		admitError := cache.Admit(review.Spec.Namespace, v1NewUsage)

		if admitError != nil {
			return &admissionv1alpha1.AdmissionReviewStatus{
				Allowed: false,
				Result: &metav1.Status{
					Message: admitError.Error(),
					Reason:  metav1.StatusReason(metav1.StatusReasonForbidden),
				},
			}
		}
	}
	return &admissionv1alpha1.AdmissionReviewStatus{
		Allowed: true,
	}
}

func internalErrorDeny(err error) *admissionv1alpha1.AdmissionReviewStatus {
	return &admissionv1alpha1.AdmissionReviewStatus{
		Allowed: false,
		Result: &metav1.Status{
			Message: err.Error(),
			Reason:  metav1.StatusReason(metav1.StatusReasonInternalError),
		},
	}
}
