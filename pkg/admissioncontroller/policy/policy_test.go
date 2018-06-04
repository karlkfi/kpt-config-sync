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

package policy

import (
	"testing"

	pn_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/testing/fakeinformers"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	"k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestAuthorize(t *testing.T) {
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

	testCases := []struct {
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
			name: "valid policynode create request",
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
						Kind:    "PolicyNode",
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
			name: "invalid create request: orphan add",
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
						Kind:    "PolicyNode",
					},
					Object: runtime.RawExtension{
						Object: runtime.Object(&pn_v1.PolicyNode{
							ObjectMeta: metav1.ObjectMeta{
								Name: "moarkitties",
							},
							Spec: pn_v1.PolicyNodeSpec{
								Parent: "does not exist",
							},
						}),
					},
					Operation: admissionv1beta1.Create,
					Namespace: "kitties",
				},
			},
			expectedReason: metav1.StatusReasonForbidden,
		},
		{
			name: "valid policynode update request",
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
						Kind:    "PolicyNode",
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
			name: "invalid policynode update request: updating node that doesn't exist",
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
						Kind:    "PolicyNode",
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
						Kind:    "PolicyNode",
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
						Kind:    "PolicyNode",
					},
					Name:      "bigkitties",
					Operation: admissionv1beta1.Delete,
					Namespace: "bigkitties",
				},
			},
			expectedAllowed: false,
			expectedReason:  metav1.StatusReasonForbidden,
		},
		{
			name: "valid clusterpolicy create request",
			request: admissionv1beta1.AdmissionReview{
				Request: &admissionv1beta1.AdmissionRequest{
					Resource: metav1.GroupVersionResource{
						Group:    "nomos.dev",
						Version:  "v1",
						Resource: "clusterpolicies",
					},
					Kind: metav1.GroupVersionKind{
						Group:   "nomos.dev",
						Version: "v1",
						Kind:    "ClusterPolicy",
					},
					Object: runtime.RawExtension{
						Object: runtime.Object(&pn_v1.ClusterPolicy{
							ObjectMeta: metav1.ObjectMeta{
								Name: pn_v1.ClusterPolicyName,
							},
						}),
					},
					Operation: admissionv1beta1.Create,
				},
			},
			expectedAllowed: true,
		},
		{
			name: "invalid clusterpolicy create request: namespace parent",
			request: admissionv1beta1.AdmissionReview{
				Request: &admissionv1beta1.AdmissionRequest{
					Resource: metav1.GroupVersionResource{
						Group:    "nomos.dev",
						Version:  "v1",
						Resource: "clusterpolicies",
					},
					Kind: metav1.GroupVersionKind{
						Group:   "nomos.dev",
						Version: "v1",
						Kind:    "ClusterPolicy",
					},
					Object: runtime.RawExtension{
						Object: runtime.Object(&pn_v1.ClusterPolicy{
							ObjectMeta: metav1.ObjectMeta{
								Name: "kitty",
							},
						}),
					},
					Operation: admissionv1beta1.Create,
				},
			},
			expectedReason: metav1.StatusReasonForbidden,
		},
		{
			name: "valid clusterpolicy update request",
			request: admissionv1beta1.AdmissionReview{
				Request: &admissionv1beta1.AdmissionRequest{
					Resource: metav1.GroupVersionResource{
						Group:    "nomos.dev",
						Version:  "v1",
						Resource: "clusterpolicies",
					},
					Kind: metav1.GroupVersionKind{
						Group:   "nomos.dev",
						Version: "v1",
						Kind:    "ClusterPolicy",
					},
					Object: runtime.RawExtension{
						Object: runtime.Object(&pn_v1.ClusterPolicy{
							ObjectMeta: metav1.ObjectMeta{
								Name: pn_v1.ClusterPolicyName,
							},
						}),
					},
					Operation: admissionv1beta1.Update,
				},
			},
			expectedAllowed: true,
		},
		{
			name: "invalid clusterpolicy update request: duplicate clusterroles",
			request: admissionv1beta1.AdmissionReview{
				Request: &admissionv1beta1.AdmissionRequest{
					Resource: metav1.GroupVersionResource{
						Group:    "nomos.dev",
						Version:  "v1",
						Resource: "clusterpolicies",
					},
					Kind: metav1.GroupVersionKind{
						Group:   "nomos.dev",
						Version: "v1",
						Kind:    "ClusterPolicy",
					},
					Object: runtime.RawExtension{
						Object: runtime.Object(&pn_v1.ClusterPolicy{
							ObjectMeta: metav1.ObjectMeta{
								Name: pn_v1.ClusterPolicyName,
							},
							Spec: pn_v1.ClusterPolicySpec{
								ClusterRolesV1: []v1.ClusterRole{
									{
										ObjectMeta: metav1.ObjectMeta{
											Name: "role",
										},
									},
									{
										ObjectMeta: metav1.ObjectMeta{
											Name: "role",
										},
									},
								},
							},
						}),
					},
					Operation: admissionv1beta1.Update,
				},
			},
			expectedAllowed: false,
			expectedReason:  metav1.StatusReasonForbidden,
		},
		{
			name: "valid clusterpolicy delete request",
			request: admissionv1beta1.AdmissionReview{
				Request: &admissionv1beta1.AdmissionRequest{
					Resource: metav1.GroupVersionResource{
						Group:    "nomos.dev",
						Version:  "v1",
						Resource: "clusterpolicies",
					},
					Kind: metav1.GroupVersionKind{
						Group:   "nomos.dev",
						Version: "v1",
						Kind:    "ClusterPolicy",
					},
					Name:      pn_v1.ClusterPolicyName,
					Operation: admissionv1beta1.Delete,
				},
			},
			expectedAllowed: true,
		},
	}

	for _, tc := range testCases {
		actual := admitter.Admit(tc.request)
		if actual.Allowed != tc.expectedAllowed ||
			(actual.Result != nil && actual.Result.Reason != tc.expectedReason) {
			t.Errorf("[%s] Expected:\n%+v\n---\nActual:\n%+v", tc.name, tc, actual)
		}
	}
}
