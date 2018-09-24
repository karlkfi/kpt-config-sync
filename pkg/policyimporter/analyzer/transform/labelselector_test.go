/*
Copyright 2018 The Nomos Authors.

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

package transform

import (
	"testing"

	"encoding/json"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	v1alpha1 "github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var simpleSelector = v1alpha1.NamespaceSelector{
	Spec: v1alpha1.NamespaceSelectorSpec{
		Selector: metav1.LabelSelector{
			MatchLabels: map[string]string{"env": "prod"}}}}

var complexSelector = v1alpha1.NamespaceSelector{
	Spec: v1alpha1.NamespaceSelectorSpec{
		Selector: metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{{Key: "privacy", Operator: metav1.LabelSelectorOpIn, Values: []string{"sensitive", "restricted"}}},
			MatchLabels:      map[string]string{"env": "prod"}}}}

type testCase struct {
	testName           string
	policy             metav1.Object
	nsLabels           map[string]string
	expectedApplicable bool
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
			policy:             createPolicy(&simpleSelector),
			nsLabels:           map[string]string{"env": "prod"},
			expectedApplicable: true,
		},
		{
			testName:           "Complex selector",
			policy:             createPolicy(&complexSelector),
			nsLabels:           map[string]string{"env": "prod", "privacy": "sensitive"},
			expectedApplicable: true,
		},
		{
			testName:           "No match",
			policy:             createPolicy(&complexSelector),
			nsLabels:           map[string]string{"env": "prod", "privacy": "open"},
			expectedApplicable: false,
		},
		{
			testName:           "No labels",
			policy:             createPolicy(&simpleSelector),
			expectedApplicable: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {

			applicable := IsPolicyApplicableToNamespace(tc.nsLabels, tc.policy)
			if tc.expectedApplicable != applicable {
				t.Fatalf("Result didn't match, expected=%t, actual=%t", tc.expectedApplicable, applicable)
			}

		})
	}
}

func createPolicy(s *v1alpha1.NamespaceSelector) metav1.Object {
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
