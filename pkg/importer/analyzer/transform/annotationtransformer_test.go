package transform

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	rbacv1 "k8s.io/api/rbac/v1"
)

type transformerTestCase struct {
	testName            string
	mapping             map[string]valueMap
	expectedAnnotations map[string]string
	expectedError       bool
}

var transformerTestCases = []transformerTestCase{
	{
		testName:            "No transform, no mapping",
		mapping:             map[string]valueMap{},
		expectedAnnotations: map[string]string{"key1": "val1", "key2": "val2"},
	},
	{
		testName: "No transform, no matches",
		mapping: map[string]valueMap{
			"key3": {"val3": "!val3"},
		},
		expectedAnnotations: map[string]string{"key1": "val1", "key2": "val2"},
	},
	{
		testName: "Single transform",
		mapping: map[string]valueMap{
			"key1": {"val1": "!val1"},
		},
		expectedAnnotations: map[string]string{"key1": "!val1", "key2": "val2"},
	},
	{
		testName: "Multiple transforms",
		mapping: map[string]valueMap{
			"key1": {"val1": "!val1"},
			"key2": {"val2": "!val2"},
		},
		expectedAnnotations: map[string]string{"key1": "!val1", "key2": "!val2"},
	},
	{
		testName: "Invalid original value",
		mapping: map[string]valueMap{
			"key1": {"val3": "!val3"},
		},
		expectedError: true,
	},
}

func TestTransformer(t *testing.T) {
	for _, tc := range transformerTestCases {
		t.Run(tc.testName, func(t *testing.T) {

			rb := rbacv1.RoleBinding{}
			rb.SetName("rb")
			rb.SetAnnotations(map[string]string{"key1": "val1", "key2": "val2"})

			at := annotationTransformer{}
			for k, v := range tc.mapping {
				at.addMappingForKey(k, v)
			}
			err := at.transform(&rb)
			if tc.expectedError {
				if err == nil {
					t.Fatalf("Expected error")
				}
				return
			} else if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			actual := rb.GetAnnotations()

			if diff := cmp.Diff(actual, tc.expectedAnnotations); diff != "" {
				t.Errorf("Actual and expected annotations didn't match: %v", diff)
			}

		})
	}
}
