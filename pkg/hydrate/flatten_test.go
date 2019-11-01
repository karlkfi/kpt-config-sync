package hydrate_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/hydrate"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"github.com/google/nomos/testing/testoutput"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestFlatten(t *testing.T) {
	testCases := []struct {
		name     string
		configs  *namespaceconfig.AllConfigs
		expected []runtime.Object
	}{
		{
			name: "nil AllConfigs",
		},
		{
			name:    "empty AllConfigs",
			configs: &namespaceconfig.AllConfigs{},
		},
		{
			name: "one CRD",
			configs: &namespaceconfig.AllConfigs{
				CRDClusterConfig: testoutput.CRDClusterConfig(
					fake.CustomResourceDefinitionObject(),
				),
			},
			expected: []runtime.Object{
				fake.CustomResourceDefinition().Object,
			},
		},
		{
			name: "one Cluster object",
			configs: &namespaceconfig.AllConfigs{
				ClusterConfig: testoutput.ClusterConfig(
					fake.ClusterRoleBindingObject(),
				),
			},
			expected: []runtime.Object{
				fake.ClusterRoleBindingObject(),
			},
		},
		{
			name: "one Namespaced object",
			configs: &namespaceconfig.AllConfigs{
				NamespaceConfigs: testoutput.NamespaceConfigs(testoutput.NamespaceConfig(
					"", "namespaces/bar", object.WithoutAnnotation(v1.SourcePathAnnotationKey),
					fake.RoleBindingObject(),
				)),
			},
			expected: []runtime.Object{
				fake.NamespaceObject("bar"),
				fake.RoleBindingObject(object.Namespace("bar")),
			},
		},
		{
			name: "two Namespaced objects",
			configs: &namespaceconfig.AllConfigs{
				NamespaceConfigs: testoutput.NamespaceConfigs(testoutput.NamespaceConfig(
					"", "namespaces/bar", object.WithoutAnnotation(v1.SourcePathAnnotationKey),
					fake.RoleBindingObject(),
				), testoutput.NamespaceConfig(
					"", "namespaces/foo", object.WithoutAnnotation(v1.SourcePathAnnotationKey),
					fake.RoleObject(),
				)),
			},
			expected: []runtime.Object{
				fake.NamespaceObject("bar"),
				fake.RoleBindingObject(object.Namespace("bar")),
				fake.NamespaceObject("foo"),
				fake.RoleObject(object.Namespace("foo")),
			},
		},
		{
			name: "one of each",
			configs: &namespaceconfig.AllConfigs{
				CRDClusterConfig: testoutput.CRDClusterConfig(
					fake.CustomResourceDefinitionObject(),
				),
				ClusterConfig: testoutput.ClusterConfig(
					fake.ClusterRoleBindingObject(),
				),
				NamespaceConfigs: testoutput.NamespaceConfigs(testoutput.NamespaceConfig(
					"", "namespaces/bar", object.WithoutAnnotation(v1.SourcePathAnnotationKey),
					fake.RoleBindingObject(),
				)),
			},
			expected: []runtime.Object{
				fake.CustomResourceDefinitionObject(),
				fake.ClusterRoleBindingObject(),
				fake.NamespaceObject("bar"),
				fake.RoleBindingObject(object.Namespace("bar")),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := hydrate.Flatten(tc.configs)

			if diff := cmp.Diff(tc.expected, actual, cmpopts.SortSlices(sortRuntimeObjects)); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func sortRuntimeObjects(x, y runtime.Object) bool {
	gvkX := x.GetObjectKind().GroupVersionKind()
	gvkY := y.GetObjectKind().GroupVersionKind()
	if gvkX.Group != gvkY.Group {
		return gvkX.Group < gvkY.Group
	}
	if gvkX.Kind != gvkY.Kind {
		return gvkX.Kind < gvkY.Kind
	}

	metaX := x.(metav1.Object)
	metaY := y.(metav1.Object)
	if metaX.GetNamespace() != metaY.GetNamespace() {
		return metaX.GetNamespace() < metaY.GetNamespace()
	}
	return metaX.GetName() < metaY.GetName()
}
