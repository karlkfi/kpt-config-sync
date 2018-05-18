/*
Copyright 2018 The Nomos Authors.
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

package policynode

import (
	"testing"

	pn_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/testing/fakeinformers"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestPolicyNodeAuthorize(t *testing.T) {
	// Initial PolicyNodes.
	policyNodes := []runtime.Object{
		&pn_v1.PolicyNode{
			ObjectMeta: metav1.ObjectMeta{
				Name: "kitties",
			},
			Spec: pn_v1.PolicyNodeSpec{
				Parent: "bigkitties",
				Type:   pn_v1.Namespace,
			},
		},
		&pn_v1.PolicyNode{
			ObjectMeta: metav1.ObjectMeta{
				Name: "bigkitties",
			},
			Spec: pn_v1.PolicyNodeSpec{
				Parent: "",
				Type:   pn_v1.Policyspace,
			},
		},
	}

	policyNodeInformer := fakeinformers.NewPolicyNodeInformer(policyNodes...)

	admitter := NewAdmitter(policyNodeInformer)

	tt := []struct {
		name            string
		request         admissionv1beta1.AdmissionReview
		expectedAllowed bool
		expectedReason  metav1.StatusReason
	}{
		{
			name:            "empty request",
			request:         admissionv1beta1.AdmissionReview{},
			expectedAllowed: true,
		},
		{
			name: "valid create request",
			request: admissionv1beta1.AdmissionReview{
				Request: &admissionv1beta1.AdmissionRequest{
					Resource: metav1.GroupVersionResource{
						Group:    "nomos.dev",
						Version:  "v1",
						Resource: "policynodes",
					},
					Kind: metav1.GroupVersionKind{
						Group:   "nomos.dev",
						Version: "v1",
						Kind:    "PolicyNodes",
					},
					Object: runtime.RawExtension{
						Object: runtime.Object(&pn_v1.PolicyNode{
							ObjectMeta: metav1.ObjectMeta{
								Name: "moarkitties",
							},
							Spec: pn_v1.PolicyNodeSpec{
								Parent: "bigkitties",
								Type:   pn_v1.Policyspace,
							},
						}),
					},
					Operation: admissionv1beta1.Create,
					Namespace: "moarkitties",
				},
			},
			expectedAllowed: true,
		},
		{
			name: "invalid create request: namespace parent",
			request: admissionv1beta1.AdmissionReview{
				Request: &admissionv1beta1.AdmissionRequest{
					Resource: metav1.GroupVersionResource{
						Group:    "nomos.dev",
						Version:  "v1",
						Resource: "policynodes",
					},
					Kind: metav1.GroupVersionKind{
						Group:   "nomos.dev",
						Version: "v1",
						Kind:    "PolicyNodes",
					},
					Object: runtime.RawExtension{
						Object: runtime.Object(&pn_v1.PolicyNode{
							ObjectMeta: metav1.ObjectMeta{
								Name: "moarkitties",
							},
							Spec: pn_v1.PolicyNodeSpec{
								Parent: "kitties",
							},
						}),
					},
					Operation: admissionv1beta1.Create,
					Namespace: "kitties",
				},
			},
			// TODO(79989196): Make this fail.
			expectedAllowed: true,
			// expectedReason:  metav1.StatusReasonForbidden,
		},
		{
			name: "valid update request",
			request: admissionv1beta1.AdmissionReview{
				Request: &admissionv1beta1.AdmissionRequest{
					Resource: metav1.GroupVersionResource{
						Group:    "nomos.dev",
						Version:  "v1",
						Resource: "policynodes",
					},
					Kind: metav1.GroupVersionKind{
						Group:   "nomos.dev",
						Version: "v1",
						Kind:    "PolicyNodes",
					},
					Object: runtime.RawExtension{
						Object: runtime.Object(&pn_v1.PolicyNode{
							ObjectMeta: metav1.ObjectMeta{
								Name: "kitties",
							},
							Spec: pn_v1.PolicyNodeSpec{
								Parent: "bigkitties",
								Type:   pn_v1.Policyspace,
							},
						}),
					},
					Operation: admissionv1beta1.Update,
					Namespace: "kitties",
				},
			},
			expectedAllowed: true,
		},
		{
			name: "invalid update request",
			request: admissionv1beta1.AdmissionReview{
				Request: &admissionv1beta1.AdmissionRequest{
					Resource: metav1.GroupVersionResource{
						Group:    "nomos.dev",
						Version:  "v1",
						Resource: "policynodes",
					},
					Kind: metav1.GroupVersionKind{
						Group:   "nomos.dev",
						Version: "v1",
						Kind:    "PolicyNodes",
					},
					Object: runtime.RawExtension{
						Object: runtime.Object(&pn_v1.PolicyNode{
							ObjectMeta: metav1.ObjectMeta{
								Name: "moarkitties",
							},
							Spec: pn_v1.PolicyNodeSpec{
								Parent: "kitties",
							},
						}),
					},
					Operation: admissionv1beta1.Update,
					Namespace: "kitties",
				},
			},
			expectedAllowed: false,
			expectedReason:  metav1.StatusReasonForbidden,
		},
		{
			name: "valid delete request",
			request: admissionv1beta1.AdmissionReview{
				Request: &admissionv1beta1.AdmissionRequest{
					Resource: metav1.GroupVersionResource{
						Group:    "nomos.dev",
						Version:  "v1",
						Resource: "policynodes",
					},
					Kind: metav1.GroupVersionKind{
						Group:   "nomos.dev",
						Version: "v1",
						Kind:    "PolicyNodes",
					},
					Name:      "kitties",
					Operation: admissionv1beta1.Delete,
					Namespace: "kitties",
				},
			},
			expectedAllowed: true,
		},
		{
			name: "invalid delete request",
			request: admissionv1beta1.AdmissionReview{
				Request: &admissionv1beta1.AdmissionRequest{
					Resource: metav1.GroupVersionResource{
						Group:    "nomos.dev",
						Version:  "v1",
						Resource: "policynodes",
					},
					Kind: metav1.GroupVersionKind{
						Group:   "nomos.dev",
						Version: "v1",
						Kind:    "PolicyNodes",
					},
					Name:      "bigkitties",
					Operation: admissionv1beta1.Delete,
					Namespace: "bigkitties",
				},
			},
			expectedAllowed: false,
			expectedReason:  metav1.StatusReasonForbidden,
		},
	}

	for _, ttt := range tt {
		actual := admitter.Admit(ttt.request)
		if actual.Allowed != ttt.expectedAllowed ||
			(actual.Result != nil && actual.Result.Reason != ttt.expectedReason) {
			t.Errorf("[%s] Expected:\n%+v\n---\nActual:\n%+v", ttt.name, ttt, actual)
		}
	}
}
