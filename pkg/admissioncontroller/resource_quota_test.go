// Reviewed by sunilarora
package admissioncontroller

import (
	"testing"

	pn_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/testing/fakeinformers"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestAuthorize(t *testing.T) {
	// Initial setup of quotas
	// Limits and structure
	policyNodes := []runtime.Object{
		&pn_v1.PolicyNode{
			ObjectMeta: metav1.ObjectMeta{
				Name: "kitties",
			},
			Spec: pn_v1.PolicyNodeSpec{
				Parent: "bigkitties",
				Policies: pn_v1.Policies{
					ResourceQuotaV1: &core_v1.ResourceQuota{Spec: core_v1.ResourceQuotaSpec{
						Hard: core_v1.ResourceList{
							"pods":    resource.MustParse("1"),
							"secrets": resource.MustParse("0"),
						},
					}},
				},
			},
		},
		&pn_v1.PolicyNode{
			ObjectMeta: metav1.ObjectMeta{
				Name: "bigkitties",
			},
			Spec: pn_v1.PolicyNodeSpec{
				Parent: "",
				Policies: pn_v1.Policies{
					ResourceQuotaV1: &core_v1.ResourceQuota{Spec: core_v1.ResourceQuotaSpec{
						Hard: core_v1.ResourceList{
							"pods":    resource.MustParse("1"),
							"secrets": resource.MustParse("0"),
						},
					}},
				},
			},
		},
	}

	policyNodeInformer := fakeinformers.NewPolicyNodeInformer(policyNodes...)
	resourceQuotaInformer := fakeinformers.NewResourceQuotaInformer()

	admitter := NewResourceQuotaAdmitter(policyNodeInformer, resourceQuotaInformer)

	pod := core_v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "mypod", Namespace: "kitties"},
		Spec:       core_v1.PodSpec{},
	}

	secret := core_v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "mysecret", Namespace: "kitties"},
	}

	tt := []struct {
		request         admissionv1beta1.AdmissionReview
		expectedAllowed bool
		expectedReason  metav1.StatusReason
	}{
		{
			request:         admissionv1beta1.AdmissionReview{},
			expectedAllowed: true,
		},
		{
			request: admissionv1beta1.AdmissionReview{
				Request: &admissionv1beta1.AdmissionRequest{
					Resource: metav1.GroupVersionResource{
						Group:    "",
						Resource: "pods",
						Version:  "v1",
					},
					Kind: metav1.GroupVersionKind{
						Group:   "",
						Version: "v1",
						Kind:    "Pod",
					},
					Object: runtime.RawExtension{
						Object: runtime.Object(&pod),
					},
					Operation: admissionv1beta1.Create,
					Namespace: "kitties",
				},
			},
			expectedAllowed: true,
		},
		{
			request: admissionv1beta1.AdmissionReview{
				Request: &admissionv1beta1.AdmissionRequest{
					Resource: metav1.GroupVersionResource{
						Group:    "",
						Resource: "secrets",
						Version:  "v1",
					},
					Kind: metav1.GroupVersionKind{
						Group:   "",
						Version: "v1",
						Kind:    "Secret",
					},
					Object: runtime.RawExtension{
						Object: runtime.Object(&secret),
					},
					Operation: admissionv1beta1.Create,
					Namespace: "kitties",
				},
			},
			expectedAllowed: false,
			expectedReason:  metav1.StatusReasonForbidden,
		},
	}

	for idx, ttt := range tt {
		actual := admitter.Admit(ttt.request)
		if actual.Allowed != ttt.expectedAllowed ||
			(actual.Result != nil && actual.Result.Reason != ttt.expectedReason) {
			t.Errorf("[T%d] Expected:\n%+v\n---\nActual:\n%+v", idx, ttt, actual)
		}
	}
}
