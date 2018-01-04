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

// Interface for a dynamic admission controller
package admission_controller

import (
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
)

// The interface for admission controller implementations
type Admitter interface {
	// Returns an admission review status based on the admission review request containing the resource being modified.
	Admit(review admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse
}
