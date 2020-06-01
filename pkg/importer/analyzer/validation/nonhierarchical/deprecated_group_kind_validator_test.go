package nonhierarchical_test

import (
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	nht "github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical/nonhierarchicaltest"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/api/extensions/v1beta1"
)

func TestDeprecatedGroupKindValidator(t *testing.T) {
	tcs := []nht.ValidatorTestCase{
		nht.Fail("deprecated Deployment",
			fake.UnstructuredAtPath(v1beta1.SchemeGroupVersion.WithKind("Deployment"),
				"namespaces/deployment.yaml")),
		nht.Fail("deprecated PodSecurityPolicy",
			fake.UnstructuredAtPath(v1beta1.SchemeGroupVersion.WithKind("PodSecurityPolicy"),
				"namespaces/deployment.yaml")),
		nht.Pass("non-deprecated ingress",
			fake.UnstructuredAtPath(v1beta1.SchemeGroupVersion.WithKind("Ingress"),
				"namespaces/ingress.yaml")),
		nht.Pass("non-deprecated Deployment",
			fake.Deployment("namespaces")),
	}

	nht.RunAll(t, nonhierarchical.DeprecatedGroupKindValidator, tcs)
}
