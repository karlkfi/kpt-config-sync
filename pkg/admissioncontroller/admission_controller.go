/*
Copyright 2017 The CSP Config Management Authors.
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

// Package admissioncontroller contains the interface for Nomos validating admission controllers
package admissioncontroller

import (
	"time"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EndpointRegistrationTimeout is the time to wait for the endpoint to come up before registering
const EndpointRegistrationTimeout = time.Second * 5

// Admitter is the interface for admission controller implementations
type Admitter interface {
	// Returns an admission review status based on the admission review request
	// containing the resource being modified.
	Admit(review admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse
}

// Deny generates a deny admission response based on an error and reason.
func Deny(reason metav1.StatusReason, err error) *admissionv1beta1.AdmissionResponse {
	return &admissionv1beta1.AdmissionResponse{
		Allowed: false,
		Result: &metav1.Status{
			Message: err.Error(),
			Reason:  reason,
		},
	}
}
