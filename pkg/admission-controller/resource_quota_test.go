package admission_controller

import (
	"testing"

	admissionv1alpha1 "k8s.io/api/admission/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/api/resource"
	core_v1 "k8s.io/api/core/v1"
	"github.com/google/stolos/pkg/testing/fakeinformers"
)

func TestAuthorize(t *testing.T) {
	// Initial setup of quotas
	// Limits and structure
	policyNodes := []runtime.Object{
		makePolicyNode("kitties", "", core_v1.ResourceList{
			"pods":    resource.MustParse("1"),
			"secrets": resource.MustParse("0"),},
		),
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
		request         admissionv1alpha1.AdmissionReview
		expectedAllowed bool
		expectedReason  metav1.StatusReason
	}{
		{
			request:         admissionv1alpha1.AdmissionReview{},
			expectedAllowed: true,
		},
		{
			request: admissionv1alpha1.AdmissionReview{
				Spec: admissionv1alpha1.AdmissionReviewSpec{
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
					Operation: admissionv1alpha1.Create,
					Namespace: "kitties",
				},
			},
			expectedAllowed: true,
		},
		{
			request: admissionv1alpha1.AdmissionReview{
				Spec: admissionv1alpha1.AdmissionReviewSpec{
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
					Operation: admissionv1alpha1.Create,
					Namespace: "kitties",
				},
			},
			expectedAllowed: false,
			expectedReason:  metav1.StatusReasonForbidden,
		},
	}

	for idx, ttt := range tt {
		actual := admitter.Admit(ttt.request)
		if actual.Allowed != ttt.expectedAllowed && actual.Result.Reason != ttt.expectedReason {
			t.Errorf("[T%d] Expected:\n%+v\n---\nActual:\n%+v", idx, ttt, actual)
		}
	}
}
