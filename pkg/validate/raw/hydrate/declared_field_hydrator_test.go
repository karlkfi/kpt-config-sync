package hydrate

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/validate/objects"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestDeclaredFields(t *testing.T) {
	converter, err := declared.ValueConverterForTest()
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name string
		objs *objects.Raw
		want *objects.Raw
	}{
		{
			name: "encode fields for Role with rules",
			objs: &objects.Raw{
				Converter: converter,
				Objects: []ast.FileObject{
					fake.FileObject(&unstructured.Unstructured{
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
					}, "role.yaml"),
				},
			},
			want: &objects.Raw{
				Converter: converter,
				Objects: []ast.FileObject{
					fake.FileObject(&unstructured.Unstructured{
						Object: map[string]interface{}{
							"apiVersion": rbacv1.SchemeGroupVersion.String(),
							"kind":       "Role",
							"metadata": map[string]interface{}{
								"name":      "hello",
								"namespace": "world",
								"annotations": map[string]interface{}{
									v1alpha1.DeclaredFieldsKey: `{"f:rules":{}}`,
								},
							},
							"rules": []interface{}{
								map[string]interface{}{
									"apiGroups": []interface{}{""},
									"resources": []interface{}{"namespaces"},
									"verbs":     []interface{}{"get", "list"},
								},
							},
						},
					}, "role.yaml"),
				},
			},
		},
		{
			name: "encode fields for Role with a label",
			objs: &objects.Raw{
				Converter: converter,
				Objects: []ast.FileObject{
					fake.FileObject(&unstructured.Unstructured{
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
					}, "role.yaml"),
				},
			},
			want: &objects.Raw{
				Converter: converter,
				Objects: []ast.FileObject{
					fake.FileObject(&unstructured.Unstructured{
						Object: map[string]interface{}{
							"apiVersion": rbacv1.SchemeGroupVersion.String(),
							"kind":       "Role",
							"metadata": map[string]interface{}{
								"name":      "hello",
								"namespace": "world",
								"labels": map[string]interface{}{
									"this": "that",
								},
								"annotations": map[string]interface{}{
									v1alpha1.DeclaredFieldsKey: `{"f:metadata":{"f:labels":{"f:this":{}}}}`,
								},
							},
						},
					}, "role.yaml"),
				},
			},
		},
		{
			name: "encode fields for Custom Resource",
			objs: &objects.Raw{
				Converter: converter,
				Objects: []ast.FileObject{
					fake.FileObject(&unstructured.Unstructured{
						Object: map[string]interface{}{
							"apiVersion": "acme.com/v1",
							"kind":       "Anvil",
							"metadata": map[string]interface{}{
								"name":        "heavy",
								"namespace":   "foo",
								"annotations": map[string]interface{}{},
								"labels":      map[string]interface{}{},
							},
							"spec": map[string]interface{}{
								"lbs": 123,
							},
						},
					}, "anvil.yaml"),
				},
			},
			want: &objects.Raw{
				Converter: converter,
				Objects: []ast.FileObject{
					fake.FileObject(&unstructured.Unstructured{
						Object: map[string]interface{}{
							"apiVersion": "acme.com/v1",
							"kind":       "Anvil",
							"metadata": map[string]interface{}{
								"name":      "heavy",
								"namespace": "foo",
								"annotations": map[string]interface{}{
									v1alpha1.DeclaredFieldsKey: `{"f:metadata":{"f:annotations":{},"f:labels":{}},"f:spec":{".":{},"f:lbs":{}}}`,
								},
							},
							"spec": map[string]interface{}{
								"lbs": 123,
							},
						},
					}, "anvil.yaml"),
				},
			},
		},
	}

	ignoreConverter := cmpopts.IgnoreFields(objects.Raw{}, "Converter")

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if errs := DeclaredFields(tc.objs); errs != nil {
				t.Errorf("Got DeclaredFields() error %v, want nil", errs)
			}
			if diff := cmp.Diff(tc.want, tc.objs, ast.CompareFileObject, ignoreConverter); diff != "" {
				t.Error(diff)
			}
		})
	}
}
