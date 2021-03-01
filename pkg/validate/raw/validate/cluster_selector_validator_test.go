package validate

import (
	"errors"
	"testing"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/validate/objects"
)

var (
	legacyClusterSelectorAnnotation = core.Annotation(v1.LegacyClusterSelectorAnnotationKey, "prod-selector")
	inlineClusterSelectorAnnotation = core.Annotation(v1alpha1.ClusterNameSelectorAnnotationKey, "prod-cluster")
)

func TestClusterSelectorsForHierarchical(t *testing.T) {
	testCases := []struct {
		name     string
		objs     *objects.Raw
		wantErrs status.MultiError
	}{
		{
			name: "No objects",
			objs: &objects.Raw{},
		},
		{
			name: "One ClusterSelector",
			objs: &objects.Raw{
				Objects: []ast.FileObject{
					fake.ClusterSelector(core.Name("first")),
				},
			},
		},
		{
			name: "Two ClusterSelectors",
			objs: &objects.Raw{
				Objects: []ast.FileObject{
					fake.ClusterSelector(core.Name("first")),
					fake.ClusterSelector(core.Name("second")),
				},
			},
		},
		{
			name: "Duplicate ClusterSelectors",
			objs: &objects.Raw{
				Objects: []ast.FileObject{
					fake.ClusterSelector(core.Name("first")),
					fake.ClusterSelector(core.Name("first")),
				},
			},
			wantErrs: nonhierarchical.SelectorMetadataNameCollisionError(kinds.ClusterSelector().Kind, "first", fake.ClusterSelector()),
		},
		{
			name: "Object with no cluster selector",
			objs: &objects.Raw{
				Objects: []ast.FileObject{
					fake.ClusterRole(),
				},
			},
		},
		{
			name: "Object with legacy cluster selector",
			objs: &objects.Raw{
				Objects: []ast.FileObject{
					fake.ClusterRole(legacyClusterSelectorAnnotation),
				},
			},
		},
		{
			name: "Object with inline cluster selector",
			objs: &objects.Raw{
				Objects: []ast.FileObject{
					fake.ClusterRole(inlineClusterSelectorAnnotation),
				},
			},
		},
		{
			name: "Non-selectable objects with no cluster selectors",
			objs: &objects.Raw{
				Objects: []ast.FileObject{
					fake.Cluster(),
					fake.ClusterSelector(),
					fake.NamespaceSelector(),
					fake.CustomResourceDefinitionV1(),
					fake.CustomResourceDefinitionV1Beta1(),
				},
			},
		},
		{
			name: "Non-selectable objects with legacy cluster selectors",
			objs: &objects.Raw{
				Objects: []ast.FileObject{
					fake.Cluster(legacyClusterSelectorAnnotation),
					fake.ClusterSelector(legacyClusterSelectorAnnotation),
					fake.NamespaceSelector(legacyClusterSelectorAnnotation),
					fake.CustomResourceDefinitionV1(legacyClusterSelectorAnnotation),
					fake.CustomResourceDefinitionV1Beta1(legacyClusterSelectorAnnotation),
				},
			},
			wantErrs: status.Append(nil,
				nonhierarchical.IllegalClusterSelectorAnnotationError(fake.Cluster(), "legacy"),
				nonhierarchical.IllegalClusterSelectorAnnotationError(fake.ClusterSelector(), "legacy"),
				nonhierarchical.IllegalClusterSelectorAnnotationError(fake.NamespaceSelector(), "legacy"),
				nonhierarchical.IllegalClusterSelectorAnnotationError(fake.CustomResourceDefinitionV1(), "legacy"),
				nonhierarchical.IllegalClusterSelectorAnnotationError(fake.CustomResourceDefinitionV1Beta1(), "legacy"),
			),
		},
		{
			name: "Non-selectable objects with inline cluster selectors",
			objs: &objects.Raw{
				Objects: []ast.FileObject{
					fake.Cluster(inlineClusterSelectorAnnotation),
					fake.ClusterSelector(inlineClusterSelectorAnnotation),
					fake.NamespaceSelector(inlineClusterSelectorAnnotation),
					fake.CustomResourceDefinitionV1(inlineClusterSelectorAnnotation),
					fake.CustomResourceDefinitionV1Beta1(inlineClusterSelectorAnnotation),
				},
			},
			wantErrs: status.Append(nil,
				nonhierarchical.IllegalClusterSelectorAnnotationError(fake.Cluster(), "inline"),
				nonhierarchical.IllegalClusterSelectorAnnotationError(fake.ClusterSelector(), "inline"),
				nonhierarchical.IllegalClusterSelectorAnnotationError(fake.NamespaceSelector(), "inline"),
				nonhierarchical.IllegalClusterSelectorAnnotationError(fake.CustomResourceDefinitionV1(), "inline"),
				nonhierarchical.IllegalClusterSelectorAnnotationError(fake.CustomResourceDefinitionV1Beta1(), "inline"),
			),
		},
		{
			name: "Cluster and legacy cluster selector in wrong directory",
			objs: &objects.Raw{
				Objects: []ast.FileObject{
					fake.ClusterAtPath("system/cluster.yaml"),
					fake.ClusterSelectorAtPath("cluster/cs.yaml"),
				},
			},
			wantErrs: status.Append(nil,
				validation.ShouldBeInClusterRegistryError(fake.Cluster()),
				validation.ShouldBeInClusterRegistryError(fake.ClusterSelector()),
			),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errs := ClusterSelectorsForHierarchical(tc.objs)
			if !errors.Is(errs, tc.wantErrs) {
				t.Errorf("got ClusterSelectorsForHierarchical() error %v, want %v", errs, tc.wantErrs)
			}
		})
	}
}

func TestClusterSelectorsForUnstructured(t *testing.T) {
	testCases := []struct {
		name     string
		objs     *objects.Raw
		wantErrs status.MultiError
	}{
		// We really just need to verify that unstructured does not care about the
		// directory of the files.
		{
			name: "Cluster and legacy cluster selector in wrong directory",
			objs: &objects.Raw{
				Objects: []ast.FileObject{
					fake.ClusterAtPath("system/cluster.yaml"),
					fake.ClusterSelectorAtPath("cluster/cs.yaml"),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errs := ClusterSelectorsForUnstructured(tc.objs)
			if !errors.Is(errs, tc.wantErrs) {
				t.Errorf("got ClusterSelectorsForUnstructured() error %v, want %v", errs, tc.wantErrs)
			}
		})
	}
}
