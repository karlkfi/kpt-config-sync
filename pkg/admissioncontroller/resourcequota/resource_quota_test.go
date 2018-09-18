package resourcequota

import (
	"testing"

	pnv1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/testing/fakeinformers"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestQuotaAuthorize(t *testing.T) {
	// Initial setup of quotas
	// Limits and structure
	policyNodes := []runtime.Object{
		&pnv1.PolicyNode{
			ObjectMeta: metav1.ObjectMeta{
				Name: "kitties",
			},
			Spec: pnv1.PolicyNodeSpec{
				Parent: "bigkitties",
				ResourceQuotaV1: &corev1.ResourceQuota{
					Spec: corev1.ResourceQuotaSpec{
						Hard: corev1.ResourceList{
							"pods":    resource.MustParse("1"),
							"secrets": resource.MustParse("0"),
						},
					},
				},
			},
		},
		&pnv1.PolicyNode{
			ObjectMeta: metav1.ObjectMeta{
				Name: "bigkitties",
			},
			Spec: pnv1.PolicyNodeSpec{
				Parent: "",
				ResourceQuotaV1: &corev1.ResourceQuota{
					Spec: corev1.ResourceQuotaSpec{
						Hard: corev1.ResourceList{
							"pods":    resource.MustParse("1"),
							"secrets": resource.MustParse("0"),
						},
					},
				},
			},
		},
	}

	policyNodeInformer := fakeinformers.NewPolicyNodeInformer(policyNodes...)
	resourceQuotaInformer := fakeinformers.NewResourceQuotaInformer()

	admitter := NewAdmitter(policyNodeInformer, resourceQuotaInformer)

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "mypod", Namespace: "kitties"},
		Spec:       corev1.PodSpec{},
	}

	secret := corev1.Secret{
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
