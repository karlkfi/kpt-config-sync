/*
Copyright 2017 The Nomos Authors.
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

package filesystem

import (
	"testing"

	"github.com/go-test/deep"
	rbac_v1 "k8s.io/api/rbac/v1"
)

type transformerTestCase struct {
	testName            string
	mapping             map[string]map[string]string
	expectedAnnotations map[string]string
	expectedError       bool
}

var transformerTestCases = []transformerTestCase{
	{
		testName:            "No transform, no mapping",
		mapping:             map[string]map[string]string{},
		expectedAnnotations: map[string]string{"key1": "val1", "key2": "val2"},
	},
	{
		testName: "No transform, no matches",
		mapping: map[string]map[string]string{
			"key3": {"val3": "!val3"},
		},
		expectedAnnotations: map[string]string{"key1": "val1", "key2": "val2"},
	},
	{
		testName: "Single transform",
		mapping: map[string]map[string]string{
			"key1": {"val1": "!val1"},
		},
		expectedAnnotations: map[string]string{"key1": "!val1", "key2": "val2"},
	},
	{
		testName: "Multiple transforms",
		mapping: map[string]map[string]string{
			"key1": {"val1": "!val1"},
			"key2": {"val2": "!val2"},
		},
		expectedAnnotations: map[string]string{"key1": "!val1", "key2": "!val2"},
	},
	{
		testName: "Invalid original value",
		mapping: map[string]map[string]string{
			"key1": {"val3": "!val3"},
		},
		expectedError: true,
	},
}

func TestTransformer(t *testing.T) {
	for _, tc := range transformerTestCases {
		t.Run(tc.testName, func(t *testing.T) {

			rb := rbac_v1.RoleBinding{}
			rb.SetName("rb")
			rb.SetAnnotations(map[string]string{"key1": "val1", "key2": "val2"})

			at := newAnnotationTransformer()
			for k, v := range tc.mapping {
				at.addMappingForKey(k, v)
			}
			rb2, err := at.transform(&rb)
			if tc.expectedError {
				if err == nil {
					t.Fatalf("Expected error")
				}
				return
			} else if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			actual := rb2.(*rbac_v1.RoleBinding).GetAnnotations()

			if diff := deep.Equal(actual, tc.expectedAnnotations); diff != nil {
				t.Fatalf("Actual and expected annotations didn't match: %v", diff)
			}

		})
	}
}
