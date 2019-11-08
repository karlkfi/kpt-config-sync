package selectors

import (
	"encoding/json"
	"testing"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/transform/selectors/seltest"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/vet/vettesting"
	rbacv1 "k8s.io/api/rbac/v1"
)

type testCase struct {
	testName           string
	config             core.Object
	nsLabels           map[string]string
	expectedApplicable bool
	errors             []string
}

func TestIsConfigApplicableToNamespacenfigApplicableToNamespace(t *testing.T) {
	testCases := []testCase{
		{
			testName:           "No annotation",
			config:             createConfig(nil),
			nsLabels:           map[string]string{"env": "prod"},
			expectedApplicable: true,
		},
		{
			testName:           "Simple selector",
			config:             createConfig(&seltest.ProdNamespaceSelector),
			nsLabels:           map[string]string{"env": "prod"},
			expectedApplicable: true,
		},
		{
			testName:           "Complex selector",
			config:             createConfig(&seltest.SensitiveNamespaceSelector),
			nsLabels:           map[string]string{"env": "prod", "privacy": "sensitive"},
			expectedApplicable: true,
		},
		{
			testName:           "No match",
			config:             createConfig(&seltest.SensitiveNamespaceSelector),
			nsLabels:           map[string]string{"env": "prod", "privacy": "open"},
			expectedApplicable: false,
		},
		{
			testName:           "No labels",
			config:             createConfig(&seltest.ProdNamespaceSelector),
			expectedApplicable: false,
		},
		{
			testName: "Unmarshallable",
			config:   createConfigAnnotation("{"),
			errors:   []string{vet.InvalidSelectorErrorCode},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			applicable, err := IsConfigApplicableToNamespace(tc.nsLabels, tc.config)
			if tc.expectedApplicable != applicable {
				t.Fatalf("Result didn't match, expected=%t, actual=%t", tc.expectedApplicable, applicable)
			}
			vettesting.ExpectErrors(tc.errors, err, t)
		})
	}
}

func createConfigAnnotation(annotation string) core.Object {
	rb := &rbacv1.RoleBinding{}
	rb.SetName("rb")
	rb.SetAnnotations(map[string]string{v1.NamespaceSelectorAnnotationKey: annotation})
	return rb
}

func createConfig(s *v1.NamespaceSelector) core.Object {
	rb := &rbacv1.RoleBinding{}
	rb.SetName("rb")
	if s != nil {
		j, err := json.Marshal(s)
		if err != nil {
			panic(err)
		}
		rb.SetAnnotations(map[string]string{v1.NamespaceSelectorAnnotationKey: string(j)})
	}
	return rb
}
