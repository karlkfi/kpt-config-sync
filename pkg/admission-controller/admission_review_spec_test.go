package admission_controller

import (
	"testing"

	admission_v1beta1 "k8s.io/api/admission/v1beta1"
	authenticationv1 "k8s.io/api/authentication/v1"
	"k8s.io/apiserver/pkg/admission"
)

func TestConversion(t *testing.T) {
	x := admission_v1beta1.AdmissionRequest{
			Name:      "ice cream",
			Operation: "DELETE",
			UserInfo: authenticationv1.UserInfo{
				Username: "taco",
				Extra: map[string]authenticationv1.ExtraValue{
					"choco": {"dark", "white"},
				},
			},
	}

	spec := AdmissionReviewSpec(x)
	attributes := admission.Attributes(&spec)

	if attributes.GetName() != "ice cream" {
		t.Error("Bad conversion for name")
	}
	if attributes.GetUserInfo().GetName() != "taco" {
		t.Error("Bad conversion for userinfo.name")
	}
	if attributes.GetUserInfo().GetExtra()["choco"][0] != "dark" {
		t.Error("Bad conversion for userinfo.extra")
	}
}
