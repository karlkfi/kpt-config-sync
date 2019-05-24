package mutate

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestRemove(t *testing.T) {
	testCases := []struct {
		name     string
		key      KeyValue
		obj      ast.FileObject
		expected ast.FileObject
	}{
		{
			name: "spec removes spec",
			key:  Key("spec"),
			obj: *ast.ParseFileObject(
				&unstructured.Unstructured{
					Object: map[string]interface{}{
						"spec": nil,
					},
				},
			),
			expected: *ast.ParseFileObject(
				&unstructured.Unstructured{
					Object: map[string]interface{}{},
				},
			),
		},
		{
			name: "spec does not remove metadata",
			key:  Key("spec"),
			obj: *ast.ParseFileObject(
				&unstructured.Unstructured{
					Object: map[string]interface{}{
						"metadata": nil,
					},
				},
			),
			expected: *ast.ParseFileObject(
				&unstructured.Unstructured{
					Object: map[string]interface{}{
						"metadata": nil,
					},
				},
			),
		},
		{
			name: "metadata namespace removes metadata namespace",
			key:  Key("metadata", "namespace"),
			obj: *ast.ParseFileObject(
				&unstructured.Unstructured{
					Object: map[string]interface{}{
						"metadata": map[string]interface{}{
							"namespace": nil,
						},
					},
				},
			),
			expected: *ast.ParseFileObject(
				&unstructured.Unstructured{
					Object: map[string]interface{}{
						"metadata": map[string]interface{}{},
					},
				},
			),
		},
		{
			name: "metadata namespace removes metadata: namespace",
			key:  Key("metadata").Value("namespace"),
			obj: *ast.ParseFileObject(
				&unstructured.Unstructured{
					Object: map[string]interface{}{
						"metadata": "namespace",
					},
				},
			),
			expected: *ast.ParseFileObject(
				&unstructured.Unstructured{
					Object: map[string]interface{}{},
				},
			),
		},
		{
			name: "metadata namespace does not removes metadata name",
			key:  Key("metadata", "namespace"),
			obj: *ast.ParseFileObject(
				&unstructured.Unstructured{
					Object: map[string]interface{}{
						"metadata": map[string]interface{}{
							"namespace": nil,
						},
					},
				},
			),
			expected: *ast.ParseFileObject(
				&unstructured.Unstructured{
					Object: map[string]interface{}{
						"metadata": map[string]interface{}{},
					},
				},
			),
		},
		{
			name: "spec finalizers kubernetes removes spec finalizers [kubernetes]",
			key:  Key("spec", "finalizers").Value("kubernetes"),
			obj: *ast.ParseFileObject(
				&unstructured.Unstructured{
					Object: map[string]interface{}{
						"spec": map[string]interface{}{
							"finalizers": []interface{}{
								"kubernetes",
							},
						},
					},
				},
			),
			expected: *ast.ParseFileObject(
				&unstructured.Unstructured{
					Object: map[string]interface{}{
						"spec": map[string]interface{}{
							// Removes entry from array inside map, but otherwise leaves array intact.
							"finalizers": []interface{}(nil),
						},
					},
				},
			),
		},
		{
			name: "spec finalizers kubernetes removes spec [finalizers] kubernetes",
			key:  Key("spec", "finalizers").Value("kubernetes"),
			obj: *ast.ParseFileObject(
				&unstructured.Unstructured{
					Object: map[string]interface{}{
						"spec": []interface{}{
							map[string]interface{}{
								"finalizers": "kubernetes",
							},
						},
					},
				},
			),
			expected: *ast.ParseFileObject(
				&unstructured.Unstructured{
					Object: map[string]interface{}{
						"spec": []interface{}{
							// Remove entry from map inside array, but otherwise leaves map intact.
							map[string]interface{}{},
						},
					},
				},
			),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.obj

			Remove(tc.key)(&actual)

			if diff := cmp.Diff(tc.expected, actual, cmp.AllowUnexported(ast.FileObject{})); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
