/*
Copyright 2018 The CSP Config Management Authors.

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

package selectors

import (
	"encoding/json"
	"testing"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/transform/selectors/seltest"
	"github.com/google/nomos/pkg/importer/analyzer/vet/vettesting"
	"github.com/google/nomos/pkg/status"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type testCase struct {
	testName           string
	policy             metav1.Object
	nsLabels           map[string]string
	expectedApplicable bool
	errors             []string
}

func TestIsPolicyApplicableToNamespace(t *testing.T) {
	testCases := []testCase{
		{
			testName:           "No annotation",
			policy:             createPolicy(nil),
			nsLabels:           map[string]string{"env": "prod"},
			expectedApplicable: true,
		},
		{
			testName:           "Simple selector",
			policy:             createPolicy(&seltest.ProdNamespaceSelector),
			nsLabels:           map[string]string{"env": "prod"},
			expectedApplicable: true,
		},
		{
			testName:           "Complex selector",
			policy:             createPolicy(&seltest.SensitiveNamespaceSelector),
			nsLabels:           map[string]string{"env": "prod", "privacy": "sensitive"},
			expectedApplicable: true,
		},
		{
			testName:           "No match",
			policy:             createPolicy(&seltest.SensitiveNamespaceSelector),
			nsLabels:           map[string]string{"env": "prod", "privacy": "open"},
			expectedApplicable: false,
		},
		{
			testName:           "No labels",
			policy:             createPolicy(&seltest.ProdNamespaceSelector),
			expectedApplicable: false,
		},
		{
			testName: "Unmarshallable",
			policy:   createPolicyAnnotation("{"),
			errors:   []string{status.UndocumentedErrorCode},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			applicable, err := IsPolicyApplicableToNamespace(tc.nsLabels, tc.policy)
			if tc.expectedApplicable != applicable {
				t.Fatalf("Result didn't match, expected=%t, actual=%t", tc.expectedApplicable, applicable)
			}
			vettesting.ExpectErrors(tc.errors, err, t)
		})
	}
}

func createPolicyAnnotation(annotation string) metav1.Object {
	rb := &rbacv1.RoleBinding{}
	rb.SetName("rb")
	rb.SetAnnotations(map[string]string{v1.NamespaceSelectorAnnotationKey: annotation})
	o, err := meta.Accessor(rb)
	if err != nil {
		panic(err)
	}
	return o
}

func createPolicy(s *v1.NamespaceSelector) metav1.Object {
	rb := &rbacv1.RoleBinding{}
	rb.SetName("rb")
	if s != nil {
		j, err := json.Marshal(s)
		if err != nil {
			panic(err)
		}
		rb.SetAnnotations(map[string]string{v1.NamespaceSelectorAnnotationKey: string(j)})
	}
	o, err := meta.Accessor(rb)
	if err != nil {
		panic(err)
	}
	return o
}
