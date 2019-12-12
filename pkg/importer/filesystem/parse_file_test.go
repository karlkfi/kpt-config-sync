package filesystem

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestParseYAMLFile(t *testing.T) {
	testCases := []struct {
		name      string
		contents  string
		expected  []*unstructured.Unstructured
		expectErr bool
	}{
		{
			name: "empty file",
		},
		{
			name: "one document",
			contents: `apiVersion: v1
kind: Namespace
metadata:
  testName: shipping
`,
			expected: []*unstructured.Unstructured{
				{
					Object: map[string]interface{}{
						"apiVersion": "v1",
						"kind":       "Namespace",
						"metadata": map[string]interface{}{
							"testName": "shipping",
						},
					},
				}},
		}, {
			name: "one document with triple-dash in a string",
			contents: `apiVersion: v1
kind: Namespace
metadata:
  testName: shipping
  labels:
    "a": "---"
`,
			expected: []*unstructured.Unstructured{
				{
					Object: map[string]interface{}{
						"apiVersion": "v1",
						"kind":       "Namespace",
						"metadata": map[string]interface{}{
							"testName": "shipping",
							"labels": map[string]interface{}{
								"a": "---",
							},
						},
					},
				}},
		},
		{
			name: "two documents",
			contents: `apiVersion: v1
kind: Namespace
metadata:
  testName: shipping
---
apiVersion: rbac/v1
kind: Role
metadata:
  testName: admin
  namespace: shipping
rules:
- apiGroups: [rbac]
  verbs: [all]
`,
			expected: []*unstructured.Unstructured{
				{
					Object: map[string]interface{}{
						"apiVersion": "v1",
						"kind":       "Namespace",
						"metadata": map[string]interface{}{
							"testName": "shipping",
						},
					},
				},
				{
					Object: map[string]interface{}{
						"apiVersion": "rbac/v1",
						"kind":       "Role",
						"metadata": map[string]interface{}{
							"testName":  "admin",
							"namespace": "shipping",
						},
						"rules": []interface{}{
							map[string]interface{}{
								"apiGroups": []interface{}{"rbac"},
								"verbs":     []interface{}{"all"},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := parseYAMLFile([]byte(tc.contents))
			if tc.expectErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			} else if err != nil {
				t.Fatal(errors.Wrap(err, "unexpected error"))
			}

			if diff := cmp.Diff(tc.expected, actual); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestParseJsonFile(t *testing.T) {
	testCases := []struct {
		name      string
		contents  string
		expected  []*unstructured.Unstructured
		expectErr bool
	}{
		{
			name: "empty file",
		},
		{
			name: "one object",
			contents: `{
  "apiVersion": "rbac/v1",
  "kind": "Role",
  "metadata": {
    "testName": "admin",
    "namespace": "shipping"
  },
  "rules": [
    {
      "apiGroups": ["rbac"],
      "verbs": ["all"]
    }
  ]
}
`,
			expected: []*unstructured.Unstructured{
				{
					Object: map[string]interface{}{
						"apiVersion": "rbac/v1",
						"kind":       "Role",
						"metadata": map[string]interface{}{
							"testName":  "admin",
							"namespace": "shipping",
						},
						"rules": []interface{}{
							map[string]interface{}{
								"apiGroups": []interface{}{"rbac"},
								"verbs":     []interface{}{"all"},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := parseJSONFile([]byte(tc.contents))

			if tc.expectErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			} else if err != nil {
				t.Fatal(errors.Wrap(err, "unexpected error"))
			}

			if diff := cmp.Diff(tc.expected, actual); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
