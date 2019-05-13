package syntax

import (
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	vt "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/api/extensions/v1beta1"
)

func TestDeprecatedGroupKindValidator(t *testing.T) {
	test := vt.ObjectValidatorTest{
		Validator: NewDeprecatedGroupKindValidator,
		ErrorCode: vet.DeprecatedGroupKindErrorCode,
		TestCases: []vt.ObjectValidatorTestCase{
			{
				Name:       "deprecated deployment",
				Object:     fake.Unstructured(v1beta1.SchemeGroupVersion.WithKind("Deployment"), Path("namespaces/deployment.yaml")),
				ShouldFail: true,
			},
			{
				Name: "deprecated podsecuritypolicy",
				Object: fake.Unstructured(v1beta1.SchemeGroupVersion.WithKind("PodSecurityPolicy"),
					Path("cluster/podsecuritypolicy.yaml")),
				ShouldFail: true,
			},
			{
				Name:   "non-deprecated ingress",
				Object: fake.Unstructured(v1beta1.SchemeGroupVersion.WithKind("Ingress"), Path("namespaces/ingress.yaml")),
			},
			{
				Name:   "non-deprecated deployment",
				Object: fake.Build(kinds.Deployment(), Path("namespaces/deployment.yaml")),
			},
		},
	}

	test.RunAll(t)
}

// Annotation adds annotation=value to the metadata.annotations of the FileObject under test.
func Path(path string) object.Mutator {
	return func(o *ast.FileObject) {
		o.Path = cmpath.FromSlash(path)
	}
}
