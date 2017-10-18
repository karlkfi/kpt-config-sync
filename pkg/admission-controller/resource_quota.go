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
	"fmt"

	"github.com/golang/glog"
	admissionv1alpha1 "k8s.io/api/admission/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	informerspolicynodev1 "github.com/google/stolos/pkg/client/informers/externalversions/k8us/v1"
	informerscorev1 "k8s.io/client-go/informers/core/v1"
)

type ResourceQuotaAdmitter struct {
	policyNodeInformer informerspolicynodev1.PolicyNodeInformer
	resourceQuotaInformer informerscorev1.ResourceQuotaInformer
}

var _ Admitter = (*ResourceQuotaAdmitter)(nil)

func NewResourceQuotaAdmitter(policyNodeInformer informerspolicynodev1.PolicyNodeInformer,
	resourceQuotaInformer informerscorev1.ResourceQuotaInformer) Admitter {
	return &ResourceQuotaAdmitter{policyNodeInformer: policyNodeInformer, resourceQuotaInformer: resourceQuotaInformer}
}

// Decides whether to admit a request
func (r* ResourceQuotaAdmitter) Admit(review admissionv1alpha1.AdmissionReview) *admissionv1alpha1.AdmissionReviewStatus {
	reviewStatus := admissionv1alpha1.AdmissionReviewStatus{
		Allowed: true,
		Result:  &metav1.Status{
			Reason: "ADMITTED, yay!",
		},
	}

	blockSecrets(review, &reviewStatus)
	glog.Infof("Admitting resource [%v], Admission result: [%v, %v], ",
		review.Spec.Kind, reviewStatus.Allowed, reviewStatus.Result.Status)
	return &reviewStatus
}

// Blocks creation of secrets. This is just an example in this sample resource quota controller to allow testing of
// DENY decisions.
func blockSecrets(review admissionv1alpha1.AdmissionReview, reviewStatus *admissionv1alpha1.AdmissionReviewStatus) {
	secretResourceType := metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}
	if review.Spec.Resource == secretResourceType && review.Spec.Operation == admissionv1alpha1.Create {
		reviewStatus.Allowed = false
		reviewStatus.Result =  &metav1.Status{
			Reason: metav1.StatusReason(fmt.Sprintf("New secrets not allowed")),
		}
	}
}
