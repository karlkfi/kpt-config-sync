package syntax

import (
	"testing"

	vt "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/api/extensions/v1beta1"
)

func TestDeprecatedGroupKindValidator(t *testing.T) {
	test := vt.ObjectValidatorTest{
		Validator: NewDeprecatedGroupKindValidator,
		ErrorCode: DeprecatedGroupKindErrorCode,
		TestCases: []vt.ObjectValidatorTestCase{
			{
				Name: "deprecated deployment",
				Object: fake.UnstructuredAtPath(v1beta1.SchemeGroupVersion.WithKind("Deployment"),
					"namespaces/deployment.yaml"),
				ShouldFail: true,
			},
			{
				Name: "deprecated podsecuritypolicy",
				Object: fake.UnstructuredAtPath(v1beta1.SchemeGroupVersion.WithKind("PodSecurityPolicy"),
					"cluster/podsecuritypolicy.yaml"),
				ShouldFail: true,
			},
			{
				Name: "non-deprecated ingress",
				Object: fake.UnstructuredAtPath(v1beta1.SchemeGroupVersion.WithKind("Ingress"),
					"namespaces/ingress.yaml"),
			},
			{
				Name:   "non-deprecated deployment",
				Object: fake.Deployment("namespaces"),
			},
		},
	}

	test.RunAll(t)
}
