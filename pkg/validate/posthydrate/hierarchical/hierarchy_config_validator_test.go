package hierarchical

import (
	"errors"
	"testing"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/hierarchyconfig"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/validate/parsed"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	missingGroup = schema.GroupVersionKind{Version: "v1", Kind: "RoleBinding"}
	missingKind  = kinds.RoleBinding().GroupVersion().WithKind("")
	unknownMode  = v1.HierarchyModeType("unknown")
)

func TestHierarchyConfigValidator(t *testing.T) {
	testCases := []struct {
		name    string
		root    parsed.Root
		wantErr status.MultiError
	}{
		{
			name: "Rolebinding allowed",
			root: &parsed.TreeRoot{
				SystemObjects: []ast.FileObject{
					fake.HierarchyConfig(
						fake.HierarchyConfigKind(v1.HierarchyModeDefault, kinds.RoleBinding())),
				},
			},
		},
		{
			name: "v1Beta1 CRD allowed",
			root: &parsed.TreeRoot{
				SystemObjects: []ast.FileObject{
					fake.HierarchyConfig(
						fake.HierarchyConfigKind(v1.HierarchyModeNone, kinds.CustomResourceDefinitionV1Beta1())),
				},
			},
		},
		{
			name: "v1 CRD allowed",
			root: &parsed.TreeRoot{
				SystemObjects: []ast.FileObject{
					fake.HierarchyConfig(
						fake.HierarchyConfigKind(v1.HierarchyModeInherit, kinds.CustomResourceDefinitionV1())),
				},
			},
		},
		{
			name: "Missing Group allowed",
			root: &parsed.TreeRoot{
				SystemObjects: []ast.FileObject{
					fake.HierarchyConfig(
						fake.HierarchyConfigKind(v1.HierarchyModeDefault, missingGroup)),
				},
			},
		},
		{
			name: "Missing Kind not allowed",
			root: &parsed.TreeRoot{
				SystemObjects: []ast.FileObject{
					fake.HierarchyConfig(
						fake.HierarchyConfigKind(v1.HierarchyModeDefault, missingKind)),
				},
			},
			wantErr: unsupportedResourceError(missingKind),
		},
		{
			name: "Namespace not allowed",
			root: &parsed.TreeRoot{
				SystemObjects: []ast.FileObject{
					fake.HierarchyConfig(
						fake.HierarchyConfigKind(v1.HierarchyModeDefault, kinds.Namespace())),
				},
			},
			wantErr: unsupportedResourceError(kinds.Namespace()),
		},
		{
			name: "configmanagement objects not allowed",
			root: &parsed.TreeRoot{
				SystemObjects: []ast.FileObject{
					fake.HierarchyConfig(
						fake.HierarchyConfigKind(v1.HierarchyModeDefault, kinds.Sync())),
				},
			},
			wantErr: unsupportedResourceError(kinds.Sync()),
		},
		{
			name: "unknown mode not allowed",
			root: &parsed.TreeRoot{
				SystemObjects: []ast.FileObject{
					fake.HierarchyConfig(
						fake.HierarchyConfigKind(unknownMode, kinds.Role())),
				},
			},
			wantErr: illegalModeError(kinds.Role(), unknownMode),
		},
	}

	for _, tc := range testCases {
		hv := HierarchyConfigValidator()
		t.Run(tc.name, func(t *testing.T) {
			err := hv(tc.root)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got HierarchyConfigValidator() error %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func illegalModeError(gvk schema.GroupVersionKind, mode v1.HierarchyModeType) status.Error {
	return hierarchyconfig.IllegalHierarchyModeError(
		hc(gvk.GroupKind(), fake.UnstructuredObject(gvk)), mode)
}

func unsupportedResourceError(gvk schema.GroupVersionKind) status.Error {
	return hierarchyconfig.UnsupportedResourceInHierarchyConfigError(
		hc(gvk.GroupKind(), fake.UnstructuredObject(gvk)))
}
