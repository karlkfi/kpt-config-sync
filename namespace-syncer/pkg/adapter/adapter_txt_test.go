package adapter

import (
	"testing"
	"strings"
	"reflect"
	"k8s.io/kubernetes/pkg/apis/rbac"
)

func TestLoad(t *testing.T) {
	nodes, err := Load("test_input.json")
	if err != nil {
		t.Errorf("Unexpected error %s", err)
	}

	expected := []PolicyNode{ PolicyNode{
		Name:"myNamespace",
		WorkingNamespace:true,
		Parent:"myFolder",
		Policies: PolicyLists {
			Roles: []rbac.Role { rbac.Role{
				Rules: []rbac.PolicyRule{{Resources:[]string{"pods"}, Verbs:[]string{"get", "watch"}}}}},
			RoleBindings: []rbac.RoleBinding{},
		},
	}, PolicyNode{Name:"myFolder", WorkingNamespace: false}}

	if !reflect.DeepEqual(nodes, expected) {
		t.Errorf("Expected %v but got %v", expected, nodes)
	}
}

func TestMissingFile(t *testing.T) {
	_, err := Load("bad_input.json")
	if !strings.Contains(err.Error(), "Error reading file:") {
		t.Errorf("Expected loading error for bad input")
	}
}
