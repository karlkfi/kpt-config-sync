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
