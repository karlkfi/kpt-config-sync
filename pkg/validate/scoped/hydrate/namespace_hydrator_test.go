package hydrate

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/validate/objects"
)

var (
	namespaceSelectorObject = fake.NamespaceSelectorObject(core.Name("dev-only"))
	namespaceSelector       = fake.FileObject(namespaceSelectorObject, "prod-only-nss.yaml")
	emptyNss                = fake.NamespaceSelector(core.Name("empty"))
	invalidNSSObject        = fake.NamespaceSelectorObject(core.Name("invalid"))
	invalidNSS              = fake.FileObject(invalidNSSObject, "invalid-nss.yaml")
)

func init() {
	namespaceSelectorObject.Spec.Selector.MatchLabels = map[string]string{
		"environment": "dev",
	}
	invalidNSSObject.Spec.Selector.MatchLabels = map[string]string{
		"environment": "xin prod",
	}
}

func TestNamespaceSelectors(t *testing.T) {
	testCases := []struct {
		name     string
		objs     *objects.Scoped
		want     *objects.Scoped
		wantErrs status.MultiError
	}{
		{
			name: "No objects",
			objs: &objects.Scoped{},
			want: &objects.Scoped{},
		},
		{
			name: "Keep object with no namespace selector",
			objs: &objects.Scoped{
				Cluster: []ast.FileObject{
					namespaceSelector,
					fake.Namespace("namespaces/prod", core.Label("environment", "prod")),
				},
				Namespace: []ast.FileObject{
					fake.Role(core.Namespace("prod")),
				},
			},
			want: &objects.Scoped{
				Cluster: []ast.FileObject{
					fake.Namespace("namespaces/prod", core.Label("environment", "prod")),
				},
				Namespace: []ast.FileObject{
					fake.Role(core.Namespace("prod")),
				},
			},
		},
		{
			name: "Copy object with active namespace selector",
			objs: &objects.Scoped{
				Cluster: []ast.FileObject{
					namespaceSelector,
					fake.Namespace("namespaces/prod", core.Label("environment", "prod")),
					fake.Namespace("namespaces/dev1", core.Label("environment", "dev")),
					fake.Namespace("namespaces/dev2", core.Label("environment", "dev")),
				},
				Namespace: []ast.FileObject{
					fake.Role(core.Annotation(v1.NamespaceSelectorAnnotationKey, "dev-only")),
				},
			},
			want: &objects.Scoped{
				Cluster: []ast.FileObject{
					fake.Namespace("namespaces/prod", core.Label("environment", "prod")),
					fake.Namespace("namespaces/dev1", core.Label("environment", "dev")),
					fake.Namespace("namespaces/dev2", core.Label("environment", "dev")),
				},
				Namespace: []ast.FileObject{
					fake.Role(
						core.Namespace("dev1"),
						core.Annotation(v1.NamespaceSelectorAnnotationKey, "dev-only")),
					fake.Role(
						core.Namespace("dev2"),
						core.Annotation(v1.NamespaceSelectorAnnotationKey, "dev-only")),
				},
			},
		},
		{
			name: "Remove object with inactive namespace selector",
			objs: &objects.Scoped{
				Cluster: []ast.FileObject{
					namespaceSelector,
					fake.Namespace("namespaces/prod", core.Label("environment", "prod")),
				},
				Namespace: []ast.FileObject{
					fake.Role(core.Annotation(v1.NamespaceSelectorAnnotationKey, "dev-only")),
				},
			},
			want: &objects.Scoped{
				Cluster: []ast.FileObject{
					fake.Namespace("namespaces/prod", core.Label("environment", "prod")),
				},
			},
		},
		{
			name: "Set default namespace on namespaced object without namespace",
			objs: &objects.Scoped{
				DefaultNamespace: "hello",
				Cluster: []ast.FileObject{
					fake.ClusterRole(),
				},
				Namespace: []ast.FileObject{
					fake.Role(),
					fake.Role(core.Namespace("world")),
				},
			},
			want: &objects.Scoped{
				DefaultNamespace: "hello",
				Cluster: []ast.FileObject{
					fake.ClusterRole(),
				},
				Namespace: []ast.FileObject{
					fake.Role(core.Namespace("hello")),
					fake.Role(core.Namespace("world")),
				},
			},
		},
		{
			name: "Error for missing namespace selector",
			objs: &objects.Scoped{
				Cluster: []ast.FileObject{
					fake.Namespace("namespaces/prod", core.Label("environment", "prod")),
				},
				Namespace: []ast.FileObject{
					fake.Role(core.Annotation(v1.NamespaceSelectorAnnotationKey, "dev-only")),
				},
			},
			want: &objects.Scoped{
				Cluster: []ast.FileObject{
					fake.Namespace("namespaces/prod", core.Label("environment", "prod")),
				},
				Namespace: []ast.FileObject{
					fake.Role(core.Annotation(v1.NamespaceSelectorAnnotationKey, "dev-only")),
				},
			},
			wantErrs: selectors.ObjectHasUnknownNamespaceSelector(fake.Role(), "dev-only"),
		},
		{
			name: "Error for empty namespace selector",
			objs: &objects.Scoped{
				Cluster: []ast.FileObject{
					emptyNss,
					fake.Namespace("namespaces/prod", core.Label("environment", "prod")),
				},
				Namespace: []ast.FileObject{
					fake.Role(core.Annotation(v1.NamespaceSelectorAnnotationKey, "empty")),
				},
			},
			want: &objects.Scoped{
				Cluster: []ast.FileObject{
					emptyNss,
					fake.Namespace("namespaces/prod", core.Label("environment", "prod")),
				},
				Namespace: []ast.FileObject{
					fake.Role(core.Annotation(v1.NamespaceSelectorAnnotationKey, "empty")),
				},
			},
			wantErrs: selectors.EmptySelectorError(emptyNss),
		},
		{
			name: "Error for invalid namespace selector",
			objs: &objects.Scoped{
				Cluster: []ast.FileObject{
					invalidNSS,
					fake.Namespace("namespaces/prod", core.Label("environment", "prod")),
				},
				Namespace: []ast.FileObject{
					fake.Role(core.Annotation(v1.NamespaceSelectorAnnotationKey, "invalid")),
				},
			},
			want: &objects.Scoped{
				Cluster: []ast.FileObject{
					invalidNSS,
					fake.Namespace("namespaces/prod", core.Label("environment", "prod")),
				},
				Namespace: []ast.FileObject{
					fake.Role(core.Annotation(v1.NamespaceSelectorAnnotationKey, "invalid")),
				},
			},
			wantErrs: selectors.InvalidSelectorError(invalidNSS, errors.New("")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errs := NamespaceSelectors(tc.objs)
			if !errors.Is(errs, tc.wantErrs) {
				t.Errorf("Got NamespaceSelectors() error %v, want %v", errs, tc.wantErrs)
			}
			if diff := cmp.Diff(tc.want, tc.objs, ast.CompareFileObject); diff != "" {
				t.Error(diff)
			}
		})
	}
}
