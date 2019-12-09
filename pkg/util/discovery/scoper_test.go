package discovery

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/kinds"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestScoper_GetScope(t *testing.T) {
	testCases := []struct {
		name      string
		scoper    Scoper
		groupKind schema.GroupKind
		expected  IsNamespaced
		expectErr bool
	}{
		{
			name:      "nil scoper returns Unknown",
			groupKind: kinds.Role().GroupKind(),
			expectErr: true,
		},
		{
			name:      "missing GroupKind returns unknown",
			scoper:    map[schema.GroupKind]IsNamespaced{},
			groupKind: kinds.Role().GroupKind(),
			expectErr: true,
		},
		{
			name: "NamespaceScope returns NamespaceScope",
			scoper: map[schema.GroupKind]IsNamespaced{
				kinds.Role().GroupKind(): NamespaceScope,
			},
			groupKind: kinds.Role().GroupKind(),
			expected:  NamespaceScope,
		},
		{
			name: "ClusterScope returns ClusterScope",
			scoper: map[schema.GroupKind]IsNamespaced{
				kinds.Role().GroupKind(): ClusterScope,
			},
			groupKind: kinds.Role().GroupKind(),
			expected:  ClusterScope,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := tc.scoper.GetGroupKindScope(tc.groupKind)

			if tc.expectErr || err != nil {
				if !tc.expectErr {
					t.Fatal("unexpected error", err)
				}
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}

			if diff := cmp.Diff(tc.expected, actual); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

const (
	group = "employees"
	kind  = "Engineer"
)

var (
	groupKind = schema.GroupKind{
		Group: group,
		Kind:  kind,
	}

	namespacedEngineer = []GroupKindScope{{groupKind, NamespaceScope}}
	globalEngineer     = []GroupKindScope{{groupKind, ClusterScope}}
)

func crd(versions ...v1beta1.CustomResourceDefinitionVersion) *v1beta1.CustomResourceDefinition {
	return &v1beta1.CustomResourceDefinition{
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group:    group,
			Versions: versions,
			Names: v1beta1.CustomResourceDefinitionNames{
				Kind: kind,
			},
		},
	}
}

func version(name string, served bool) v1beta1.CustomResourceDefinitionVersion {
	return v1beta1.CustomResourceDefinitionVersion{
		Name:   name,
		Served: served,
	}
}

func TestScopesFromCRD(t *testing.T) {

	testCases := []struct {
		name     string
		crd      *v1beta1.CustomResourceDefinition
		expected []GroupKindScope
	}{
		// Trivial cases.
		{
			name: "no versions returns empty",
			crd:  crd(),
		},
		// Test that scope is set correctly.
		{
			name: "with version returns scope",
			crd: &v1beta1.CustomResourceDefinition{
				Spec: v1beta1.CustomResourceDefinitionSpec{
					Group:   group,
					Version: "v1",
					Scope:   v1beta1.NamespaceScoped,
					Names: v1beta1.CustomResourceDefinitionNames{
						Kind: kind,
					},
				},
			},
			expected: namespacedEngineer,
		},
		{
			name: "without scope defaults to Namespaced",
			crd: &v1beta1.CustomResourceDefinition{
				Spec: v1beta1.CustomResourceDefinitionSpec{
					Group:   group,
					Version: "v1",
					Names: v1beta1.CustomResourceDefinitionNames{
						Kind: kind,
					},
				},
			},
			expected: namespacedEngineer,
		},
		{
			name: "Cluster scope if specified",
			crd: &v1beta1.CustomResourceDefinition{
				Spec: v1beta1.CustomResourceDefinitionSpec{
					Group:   group,
					Version: "v1",
					Scope:   v1beta1.ClusterScoped,
					Names: v1beta1.CustomResourceDefinitionNames{
						Kind: kind,
					},
				},
			},
			expected: globalEngineer,
		},
		// Served version conditions.
		{
			name: "with unserved version returns empty",
			crd:  crd(version("v1beta1", false)),
		},
		{
			name:     "with served version returns nonempty",
			crd:      crd(version("v1beta1", true)),
			expected: namespacedEngineer,
		},
		{
			name: "with no served versions returns empty",
			crd:  crd(version("v1beta1", false), version("v1", false)),
		},
		{
			name:     "with first version served returns nonempty",
			crd:      crd(version("v1beta1", true), version("v1", false)),
			expected: namespacedEngineer,
		},
		{
			name:     "with second served version returns nonempty",
			crd:      crd(version("v1beta1", false), version("v1", true)),
			expected: namespacedEngineer,
		},
		{
			name:     "with two served versions returns nonempty",
			crd:      crd(version("v1beta1", true), version("v1", true)),
			expected: namespacedEngineer,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := ScopesFromCRDs([]*v1beta1.CustomResourceDefinition{tc.crd})

			if diff := cmp.Diff(tc.expected, actual); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
