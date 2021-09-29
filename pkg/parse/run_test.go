package parse

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/metadata"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestSplitObjects(t *testing.T) {
	testCases := []struct {
		name             string
		objs             []ast.FileObject
		knownScopeObjs   []ast.FileObject
		unknownScopeObjs []ast.FileObject
	}{
		{
			name: "no unknown scope objects",
			objs: []ast.FileObject{
				fake.Namespace("namespaces/prod", core.Label("environment", "prod")),
				fake.Role(core.Namespace("prod")),
			},
			knownScopeObjs: []ast.FileObject{
				fake.Namespace("namespaces/prod", core.Label("environment", "prod")),
				fake.Role(core.Namespace("prod")),
			},
		},
		{
			name: "has unknown scope objects",
			objs: []ast.FileObject{
				fake.ClusterRole(
					core.Annotation(metadata.UnknownScopeAnnotationKey, metadata.UnknownScopeAnnotationValue),
				),
				fake.Namespace("namespaces/prod", core.Label("environment", "prod")),
				fake.Role(core.Namespace("prod")),
			},
			knownScopeObjs: []ast.FileObject{
				fake.Namespace("namespaces/prod", core.Label("environment", "prod")),
				fake.Role(core.Namespace("prod")),
			},
			unknownScopeObjs: []ast.FileObject{
				fake.ClusterRole(
					core.Annotation(metadata.UnknownScopeAnnotationKey, metadata.UnknownScopeAnnotationValue),
				),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotKnownScopeObjs, gotUnknownScopeObjs := splitObjects(tc.objs)
			if diff := cmp.Diff(tc.knownScopeObjs, gotKnownScopeObjs, ast.CompareFileObject); diff != "" {
				t.Errorf("cmp.Diff(tc.knownScopeObjs, gotKnownScopeObjs) = %v", diff)
			}
			if diff := cmp.Diff(tc.unknownScopeObjs, gotUnknownScopeObjs, ast.CompareFileObject); diff != "" {
				t.Errorf("cmp.Diff(tc.unknownScopeObjs, gotUnknownScopeObjs) = %v", diff)
			}
		})
	}
}
