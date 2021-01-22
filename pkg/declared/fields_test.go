package declared

import (
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestFieldConverter_EncodeDeclaredFields(t *testing.T) {
	testCases := []struct {
		name string
		obj  runtime.Object
		want string
	}{
		{
			name: "Encode a Role with a rule",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": rbacv1.SchemeGroupVersion.String(),
					"kind":       "Role",
					"metadata": map[string]interface{}{
						"name":      "hello",
						"namespace": "world",
					},
					"rules": []interface{}{
						map[string]interface{}{
							"apiGroups": []interface{}{""},
							"resources": []interface{}{"namespaces"},
							"verbs":     []interface{}{"get", "list"},
						},
					},
				},
			},
			want: `{"f:rules":{}}`,
		},
		{
			name: "Encode a Role with a label",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": rbacv1.SchemeGroupVersion.String(),
					"kind":       "Role",
					"metadata": map[string]interface{}{
						"name":      "hello",
						"namespace": "world",
						"labels": map[string]interface{}{
							"this": "that",
						},
					},
				},
			},
			want: `{"f:metadata":{"f:labels":{"f:this":{}}}}`,
		},
		{
			name: "Encode a custom resource",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "acme.com/v1",
					"kind":       "Anvil",
					"metadata": map[string]interface{}{
						"name":      "heavy",
						"namespace": "foo",
					},
					"spec": map[string]interface{}{
						"lbs": 123,
					},
				},
			},
			want: `{"f:spec":{".":{},"f:lbs":{}}}`,
		},
	}

	vc, err := ValueConverterForTest()
	if err != nil {
		t.Fatalf("Failed to create ValueConverter: %v", err)
	}
	fc := &FieldConverter{vc}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := fc.EncodeDeclaredFields(tc.obj)
			if err != nil {
				t.Errorf("Got unexpected error: %v", err)
			}
			if string(got) != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}
