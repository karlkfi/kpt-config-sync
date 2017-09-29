package admission_controller

import(
	"reflect"
	"testing"

	admissionv1alpha1 "k8s.io/api/admission/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAuthorize(t *testing.T) {
	allowed := admissionv1alpha1.AdmissionReviewStatus{
		Allowed: true,
		Result:  &metav1.Status{
			Reason: "ADMITTED, yay!",
		},
	}

	tt := []struct{
		request  admissionv1alpha1.AdmissionReview
		expected admissionv1alpha1.AdmissionReviewStatus
	}{
		{
			request: admissionv1alpha1.AdmissionReview{},
			expected: allowed,
		},
		{
			request: admissionv1alpha1.AdmissionReview{
				Spec: admissionv1alpha1.AdmissionReviewSpec{
					Resource: metav1.GroupVersionResource{
						Group: "",
						Resource: "pods",
						Version: "v1",
					},
					Operation: admissionv1alpha1.Create,
				},
			},
			expected: allowed,
		},
		{
			request: admissionv1alpha1.AdmissionReview{
				Spec: admissionv1alpha1.AdmissionReviewSpec{
					Resource: metav1.GroupVersionResource{
						Group: "",
						Resource: "secrets",
						Version: "v1",
					},
					Operation: admissionv1alpha1.Create,
				},
			},
			expected: admissionv1alpha1.AdmissionReviewStatus{
				Allowed: false,
				Result:  &metav1.Status{
					Reason: "New secrets not allowed",
				},
			},
		},
	}

	for _, ttt := range tt {
		admitter := ResourceQuotaAdmitter{}
		actual := admitter.Admit(ttt.request)
		if !reflect.DeepEqual(*actual, ttt.expected) {
			t.Errorf("Expected:\n%+v\n---\nActual:\n%+v", ttt, actual)
		}
	}
}
