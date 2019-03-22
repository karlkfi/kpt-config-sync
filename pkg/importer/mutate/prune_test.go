package mutate

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestPrune(t *testing.T) {
	testCases := []struct {
		name     string
		obj      ast.FileObject
		expected ast.FileObject
	}{
		{
			name: "removes empty map",
			obj: ast.FileObject{
				Object: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"spec": nil,
					},
				},
			},
			expected: ast.FileObject{
				Object: &unstructured.Unstructured{
					Object: map[string]interface{}{},
				},
			},
		},
		{
			name: "keeps non-empty map",
			obj: ast.FileObject{
				Object: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"spec": "finalizers",
					},
				},
			},
			expected: ast.FileObject{
				Object: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"spec": "finalizers",
					},
				},
			},
		},
		{
			name: "recursively removes empty maps",
			obj: ast.FileObject{
				Object: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"spec": map[string]interface{}{
							"finalizers": nil,
						},
					},
				},
			},
			expected: ast.FileObject{
				Object: &unstructured.Unstructured{
					Object: map[string]interface{}{},
				},
			},
		},
		{
			name: "removes empty array",
			obj: ast.FileObject{
				Object: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"spec": []interface{}{},
					},
				},
			},
			expected: ast.FileObject{
				Object: &unstructured.Unstructured{
					Object: map[string]interface{}{},
				},
			},
		},
		{
			name: "removes empty map inside array",
			obj: ast.FileObject{
				Object: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"spec": []interface{}{
							map[string]interface{}{},
						},
					},
				},
			},
			expected: ast.FileObject{
				Object: &unstructured.Unstructured{
					Object: map[string]interface{}{},
				},
			},
		},
		{
			name: "keeps non-empty array",
			obj: ast.FileObject{
				Object: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"spec": []interface{}{
							"finalizers",
						},
					},
				},
			},
			expected: ast.FileObject{
				Object: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"spec": []interface{}{
							"finalizers",
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.obj

			Prune()(&actual)

			if diff := cmp.Diff(tc.expected, actual); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
