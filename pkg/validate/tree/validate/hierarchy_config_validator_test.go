package validate

import (
	"errors"
	"testing"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/hierarchyconfig"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/validate/objects"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	missingGroup = schema.GroupVersionKind{Version: "v1", Kind: "RoleBinding"}
	missingKind  = kinds.RoleBinding().GroupVersion().WithKind("")
	unknownMode  = v1.HierarchyModeType("unknown")
)

func TestHierarchyConfig(t *testing.T) {
	testCases := []struct {
		name     string
		objs     *objects.Tree
		wantErrs status.MultiError
	}{
		{
			name: "Rolebinding allowed",
			objs: &objects.Tree{
				Cluster: []ast.FileObject{
					fake.ClusterRoleBinding(),
				},
				HierarchyConfigs: []ast.FileObject{
					fake.HierarchyConfig(
						fake.HierarchyConfigKind(v1.HierarchyModeDefault, kinds.RoleBinding())),
				},
			},
		},
		{
			name: "Missing Group allowed",
			objs: &objects.Tree{
				Cluster: []ast.FileObject{
					fake.ClusterRoleBinding(),
				},
				HierarchyConfigs: []ast.FileObject{
					fake.HierarchyConfig(
						fake.HierarchyConfigKind(v1.HierarchyModeDefault, missingGroup)),
				},
			},
		},
		{
			name: "Missing Kind not allowed",
			objs: &objects.Tree{
				Cluster: []ast.FileObject{
					fake.ClusterRoleBinding(),
				},
				HierarchyConfigs: []ast.FileObject{
					fake.HierarchyConfig(
						fake.HierarchyConfigKind(v1.HierarchyModeDefault, missingKind)),
				},
			},
			wantErrs: fake.Errors(hierarchyconfig.UnsupportedResourceInHierarchyConfigErrorCode),
		},
		{
			name: "Cluster-scoped objects not allowed",
			objs: &objects.Tree{
				Cluster: []ast.FileObject{
					fake.ClusterRoleBinding(),
				},
				HierarchyConfigs: []ast.FileObject{
					fake.HierarchyConfig(
						fake.HierarchyConfigKind(v1.HierarchyModeDefault, kinds.ClusterRoleBinding())),
				},
			},
			wantErrs: fake.Errors(hierarchyconfig.ClusterScopedResourceInHierarchyConfigErrorCode),
		},
		{
			name: "ConfigManagement objects not allowed",
			objs: &objects.Tree{
				Cluster: []ast.FileObject{
					fake.ClusterRoleBinding(),
				},
				HierarchyConfigs: []ast.FileObject{
					fake.HierarchyConfig(
						fake.HierarchyConfigKind(v1.HierarchyModeDefault, kinds.Sync())),
				},
			},
			wantErrs: fake.Errors(hierarchyconfig.UnsupportedResourceInHierarchyConfigErrorCode),
		},
		{
			name: "Unknown mode not allowed",
			objs: &objects.Tree{
				Cluster: []ast.FileObject{
					fake.ClusterRoleBinding(),
				},
				HierarchyConfigs: []ast.FileObject{
					fake.HierarchyConfig(
						fake.HierarchyConfigKind(unknownMode, kinds.Role())),
				},
			},
			wantErrs: fake.Errors(hierarchyconfig.IllegalHierarchyModeErrorCode),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errs := HierarchyConfig(tc.objs)
			if !errors.Is(errs, tc.wantErrs) {
				t.Errorf("got HierarchyConfig() error %v, want %v", errs, tc.wantErrs)
			}
		})
	}
}
