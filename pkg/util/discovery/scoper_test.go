package discovery

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/kinds"
	"github.com/pkg/errors"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestScoper_GetScope(t *testing.T) {
	testCases := []struct {
		name         string
		scopes       map[schema.GroupKind]ScopeType
		groupKind    schema.GroupKind
		errOnUnknown bool
		want         ScopeType
		wantErr      bool
	}{
		{
			name:         "nil scoper returns Unknown and error",
			errOnUnknown: true,
			groupKind:    kinds.Role().GroupKind(),
			want:         UnknownScope,
			wantErr:      true,
		},
		{
			name:         "nil scoper returns Unknown and no error",
			errOnUnknown: false,
			groupKind:    kinds.Role().GroupKind(),
			want:         UnknownScope,
			wantErr:      false,
		},
		{
			name:         "missing GroupKind returns unknown",
			scopes:       map[schema.GroupKind]ScopeType{},
			errOnUnknown: true,
			groupKind:    kinds.Role().GroupKind(),
			want:         UnknownScope,
			wantErr:      true,
		},
		{
			name: "NamespaceScope returns NamespaceScope",
			scopes: map[schema.GroupKind]ScopeType{
				kinds.Role().GroupKind(): NamespaceScope,
			},
			errOnUnknown: true,
			groupKind:    kinds.Role().GroupKind(),
			want:         NamespaceScope,
		},
		{
			name: "ClusterScope returns ClusterScope",
			scopes: map[schema.GroupKind]ScopeType{
				kinds.Role().GroupKind(): ClusterScope,
			},
			errOnUnknown: true,
			groupKind:    kinds.Role().GroupKind(),
			want:         ClusterScope,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			scoper := NewScoper(tc.scopes, tc.errOnUnknown)
			got, gotErr := scoper.GetGroupKindScope(tc.groupKind)

			if got != tc.want {
				t.Errorf("got GetGroupKindScope() = %q, want %q", got, tc.want)
			}

			var wantErr error
			if tc.wantErr {
				wantErr = UnknownGroupKindError(kinds.Namespace().GroupKind())
			}
			if !errors.Is(gotErr, wantErr) {
				t.Errorf("got GetGroupKindScope() error = %v, want %v",
					gotErr, wantErr)
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
