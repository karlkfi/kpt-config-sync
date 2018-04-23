package rolebindings

import (
	"fmt"
	"log"
	"testing"

	"github.com/google/nomos/pkg/cli"
	clitesting "github.com/google/nomos/pkg/cli/testing"
	fakemeta "github.com/google/nomos/pkg/client/meta/fake"
	apirbac "k8s.io/api/rbac/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	roleTypeMeta = meta.TypeMeta{
		Kind: "Role",
	}
	roleBindingTypeMeta = meta.TypeMeta{
		Kind: "RoleBinding",
	}
)

// Testing Roles and RoleBindings display is identical: the only difference is
// the object typing.  toRuntimeObject converts each object into its appropriate
// type and testFunc wraps the concrete call that needs to be made to test
// the appropriate type (since the methods to read are different).
func RunExample(
	toRuntimeObject func(name, namespace string) runtime.Object,
	testFunc func(ctx *cli.CommandContext) error) {
	tests := []struct {
		storage   []runtime.Object
		namespace string
		err       error
	}{
		{
			// Basic test.
			storage: []runtime.Object{
				clitesting.NewNamespace("default", ""),
				toRuntimeObject("quota1", "default"),
			},
			namespace: "default",
		},
		{
			// Only slightly more complicated.
			storage: []runtime.Object{
				clitesting.NewNamespace("root", ""),
				clitesting.NewNamespace("ns1", "root"),
				clitesting.NewNamespace("ns2", "root"),
				toRuntimeObject("quota1", "root"),
				toRuntimeObject("quota2", "root"),
				toRuntimeObject("quota3", "ns1"),
				toRuntimeObject("quota4", "ns2"),
			},
			namespace: "ns1",
		},
	}

	for i, test := range tests {
		ctx := &cli.CommandContext{
			Client:    fakemeta.NewClientWithStorage(test.storage),
			Namespace: test.namespace,
		}
		fmt.Printf("---\nTest case: %v\n", i)
		err := testFunc(ctx)
		if err != nil {
			if test.err == nil {
				log.Printf("[%v] unexpected error: %v", i, err)
			}
		}
	}
}

// Used just below.
func roleBinding(name, namespace string) runtime.Object {
	return &apirbac.RoleBinding{
		TypeMeta: roleBindingTypeMeta,
		ObjectMeta: meta.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func TestExamplesRoleBindings(_ *testing.T) {
	RunExample(roleBinding, func(ctx *cli.CommandContext) error {
		err := GetHierarchicalRoleBindings(ctx, []string{""})
		return err
	})
	// Output:
	// ---
	// Test case: 0
	// # Namespace: "default"
	// #
	// items:
	// - kind: RoleBinding
	//   metadata:
	//     creationTimestamp: null
	//     name: quota1
	//     namespace: default
	//   roleRef:
	//     apiGroup: ""
	//     kind: ""
	//     name: ""
	//   subjects: null
	// metadata: {}
	// ---
	// Test case: 1
	// # Namespace: "ns1"
	// #
	// items:
	// - kind: RoleBinding
	//   metadata:
	//     creationTimestamp: null
	//     name: quota3
	//     namespace: ns1
	//   roleRef:
	//     apiGroup: ""
	//     kind: ""
	//     name: ""
	//   subjects: null
	// metadata: {}
	// # Namespace: "root"
	// #
	// items:
	// - kind: RoleBinding
	//   metadata:
	//     creationTimestamp: null
	//     name: quota1
	//     namespace: root
	//   roleRef:
	//     apiGroup: ""
	//     kind: ""
	//     name: ""
	//   subjects: null
	// - kind: RoleBinding
	//   metadata:
	//     creationTimestamp: null
	//     name: quota2
	//     namespace: root
	//   roleRef:
	//     apiGroup: ""
	//     kind: ""
	//     name: ""
	//   subjects: null
	// metadata: {}
}

// Use just below.
func role(name, namespace string) runtime.Object {
	return &apirbac.Role{
		TypeMeta: roleTypeMeta,
		ObjectMeta: meta.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func TestExamplesRoles(_ *testing.T) {
	RunExample(role, func(ctx *cli.CommandContext) error {
		err := GetHierarchicalRoles(ctx, []string{""})
		return err
	})
	// Output:
	// ---
	// Test case: 0
	// # Namespace: "default"
	// #
	// items:
	// - kind: Role
	//   metadata:
	//     creationTimestamp: null
	//     name: quota1
	//     namespace: default
	//   rules: null
	// metadata: {}
	// ---
	// Test case: 1
	// # Namespace: "ns1"
	// #
	// items:
	// - kind: Role
	//   metadata:
	//     creationTimestamp: null
	//     name: quota3
	//     namespace: ns1
	//   rules: null
	// metadata: {}
	// # Namespace: "root"
	// #
	// items:
	// - kind: Role
	//   metadata:
	//     creationTimestamp: null
	//     name: quota1
	//     namespace: root
	//   rules: null
	// - kind: Role
	//   metadata:
	//     creationTimestamp: null
	//     name: quota2
	//     namespace: root
	//   rules: null
	// metadata: {}
}
