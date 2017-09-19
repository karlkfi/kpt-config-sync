package adapter

import (
	"reflect"
	"strings"
	"testing"

	"github.com/mdruskin/kubernetes-enterprise-control/pkg/api/policyhierarchy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

func TestLoad(t *testing.T) {
	nodes, err := Load("test_input.json")
	if err != nil {
		t.Errorf("Unexpected error %s", err)
	}

	expected := []v1.PolicyNode{
		v1.PolicyNode{
			Spec: v1.PolicyNodeSpec{

				Name:             "myNamespace",
				WorkingNamespace: true,
				Parent:           "myFolder",
				Policies: v1.PolicyLists{
					Roles: []rbacv1.Role{
						rbacv1.Role{
							Rules: []rbacv1.PolicyRule{
								{
									Resources: []string{"pods"},
									Verbs:     []string{"get", "watch"},
								},
							},
						},
					},
					RoleBindings: []rbacv1.RoleBinding{},
				},
			},
		},
		v1.PolicyNode{
			Spec: v1.PolicyNodeSpec{
				Name:             "myFolder",
				WorkingNamespace: false,
			},
		},
	}

	if !reflect.DeepEqual(nodes, expected) {
		t.Errorf("Expected:\n%+v\nbut got\n%+v", expected, nodes)
	}
}

func TestMissingFile(t *testing.T) {
	_, err := Load("bad_input.json")
	if !strings.Contains(err.Error(), "Error reading file:") {
		t.Errorf("Expected loading error for bad input")
	}
}
