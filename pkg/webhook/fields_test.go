package webhook

import (
	"testing"

	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/testing/fake"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func setRules(rules []rbacv1.PolicyRule) core.MetaMutator {
	return func(o core.Object) {
		role := o.(*rbacv1.Role)
		role.Rules = rules
	}
}

func TestObjectDiffer_Structured(t *testing.T) {
	testCases := []struct {
		name string
		muts []core.MetaMutator
		want string
	}{
		{
			name: "No changes",
			muts: []core.MetaMutator{},
			want: "",
		},
		{
			name: "Add a label",
			muts: []core.MetaMutator{
				core.Labels(map[string]string{
					"this": "that",
					"here": "there",
				}),
			},
			want: "",
		},
		{
			name: "Change a label",
			muts: []core.MetaMutator{
				core.Labels(map[string]string{
					"this": "is not that",
				}),
			},
			want: ".metadata.labels.this",
		},
		{
			name: "Remove a label",
			muts: []core.MetaMutator{
				core.Labels(map[string]string{}),
			},
			want: ".metadata.labels.this",
		},
		{
			name: "Add a rule",
			muts: []core.MetaMutator{
				setRules([]rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"namespaces"},
						Verbs:     []string{"get", "list"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"pods"},
						Verbs:     []string{"get"},
					},
				}),
			},
			want: ".rules",
		},
		{
			name: "Change a rule",
			muts: []core.MetaMutator{
				setRules([]rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"namespaces"},
						Verbs:     []string{"get", "list", "delete"},
					},
				}),
			},
			want: ".rules",
		},
		{
			name: "Remove a rule",
			muts: []core.MetaMutator{
				setRules([]rbacv1.PolicyRule{}),
			},
			want: ".rules",
		},
	}

	vc, err := declared.ValueConverterForTest()
	if err != nil {
		t.Fatalf("Failed to create ValueConverter: %v", err)
	}
	od := &ObjectDiffer{vc}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			oldObj := roleForTest()
			newObj := roleForTest(tc.muts...)
			got, err := od.DeclaredFieldDiff(oldObj, newObj)
			if err != nil {
				t.Errorf("Got unexpected error: %v", err)
			} else if got.String() != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func roleForTest(muts ...core.MetaMutator) *rbacv1.Role {
	role := fake.RoleObject(
		core.Name("hello"),
		core.Namespace("world"),
		core.Label("this", "that"),
		core.Annotation(v1alpha1.DeclaredFieldsKey, "{\"f:metadata\":{\"f:labels\":{\"f:this\":{}}},\"f:rules\":{}}"))

	role.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"namespaces"},
			Verbs:     []string{"get", "list"},
		},
	}
	for _, mut := range muts {
		mut(role)
	}
	return role
}

func TestObjectDiffer_Unstructured(t *testing.T) {
	testCases := []struct {
		name string
		muts []mutator
		want string
	}{
		{
			name: "No changes",
			muts: []mutator{},
			want: "",
		},
		{
			name: "Add a label",
			muts: []mutator{
				setLabels(t, map[string]interface{}{
					"this": "that",
					"here": "there",
				}),
			},
			want: "",
		},
		{
			name: "Change a label",
			muts: []mutator{
				setLabels(t, map[string]interface{}{
					"this": "is not that",
				}),
			},
			want: ".metadata.labels.this",
		},
		{
			name: "Remove a label",
			muts: []mutator{
				setLabels(t, map[string]interface{}{}),
			},
			want: ".metadata.labels.this",
		},
	}

	vc, err := declared.ValueConverterForTest()
	if err != nil {
		t.Fatalf("Failed to create ValueConverter: %v", err)
	}
	od := &ObjectDiffer{vc}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			oldObj := unstructuredForTest()
			newObj := unstructuredForTest(tc.muts...)
			got, err := od.DeclaredFieldDiff(oldObj, newObj)
			if err != nil {
				t.Errorf("Got unexpected error: %v", err)
			} else if got.String() != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

type mutator func(u *unstructured.Unstructured)

func setLabels(t *testing.T, labels map[string]interface{}) mutator {
	return func(u *unstructured.Unstructured) {
		t.Helper()
		err := unstructured.SetNestedMap(u.Object, labels, "metadata", "labels")
		if err != nil {
			t.Fatal(err)
		}
	}
}

func unstructuredForTest(muts ...mutator) *unstructured.Unstructured {
	u := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": rbacv1.SchemeGroupVersion.String(),
			"kind":       "Role",
			"metadata": map[string]interface{}{
				"name":      "hello",
				"namespace": "world",
				"annotations": map[string]interface{}{
					v1alpha1.DeclaredFieldsKey: "{\"f:metadata\":{\"f:labels\":{\"f:this\":{}}},\"f:rules\":{}}",
				},
				"labels": map[string]interface{}{
					"this": "that",
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
	}
	for _, mut := range muts {
		mut(u)
	}
	return u
}
