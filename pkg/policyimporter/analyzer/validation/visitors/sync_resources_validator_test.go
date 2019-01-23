package visitors

import (
	"testing"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/asttesting"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/node"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet/vettesting"
	"github.com/google/nomos/pkg/policyimporter/filesystem"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	"github.com/google/nomos/pkg/util/multierror"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// This would be in a library as part of a separate CL.
func buildTreeWithObjects(dirInfos map[string][]ast.FileObject, t *testing.T) *ast.TreeNode {
	tree := filesystem.NewDirectoryTree()
	for dir, objects := range dirInfos {
		n := tree.AddDir(nomospath.NewFakeRelative(dir), node.AbstractNamespace)
		nsObjects := make([]*ast.NamespaceObject, len(objects))
		for i, object := range objects {
			nsObjects[i] = &ast.NamespaceObject{FileObject: object}
		}
		n.Objects = nsObjects
	}
	eb := multierror.Builder{}
	result := tree.Build(&eb)
	if eb.HasErrors() {
		t.Fatal(eb.Build())
	}
	return result
}

// This would be in a library as part of a separate CL.
func buildSystem(syncs map[schema.GroupVersionKind]bool) *ast.System {
	var objects []*ast.SystemObject
	for gvk := range syncs {
		sync := &v1alpha1.Sync{
			Spec: v1alpha1.SyncSpec{
				Groups: []v1alpha1.SyncGroup{
					{
						Group: gvk.Group,
						Kinds: []v1alpha1.SyncKind{
							{
								Kind: gvk.Kind,
								Versions: []v1alpha1.SyncVersion{
									{
										Version: gvk.Version,
									},
								},
							},
						},
					},
				},
			},
		}
		objects = append(objects, &ast.SystemObject{FileObject: ast.FileObject{Object: sync}})
	}

	return &ast.System{
		Objects: objects,
	}
}

func TestSyncResourcesValidator(t *testing.T) {
	testCases := []struct {
		name       string
		syncs      map[schema.GroupVersionKind]bool
		dirInfos   map[string][]ast.FileObject
		shouldFail bool
	}{
		{
			name: "empty",
		},
		{
			name:  "role sync and role object",
			syncs: map[schema.GroupVersionKind]bool{kinds.Role(): true},
			dirInfos: map[string][]ast.FileObject{
				"namespaces": {asttesting.NewFakeFileObject(kinds.Role(), "namespaces/r.yaml")},
			},
		},
		{
			name: "missing role sync",
			dirInfos: map[string][]ast.FileObject{
				"namespaces": {asttesting.NewFakeFileObject(kinds.Role(), "namespaces/r.yaml")},
			},
			shouldFail: true,
		},
		{
			name:  "rolebinding sync and role object",
			syncs: map[schema.GroupVersionKind]bool{kinds.RoleBinding(): true},
			dirInfos: map[string][]ast.FileObject{
				"namespaces": {asttesting.NewFakeFileObject(kinds.Role(), "namespaces/r.yaml")},
			},
			shouldFail: true,
		},
		{
			name: "unsyncable rolebinding in child of namespaces",
			dirInfos: map[string][]ast.FileObject{
				"namespaces":     {},
				"namespaces/foo": {asttesting.NewFakeFileObject(kinds.Role(), "namespaces/foo/r.yaml")},
			},
			shouldFail: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			root := ast.Root{
				Tree:   buildTreeWithObjects(tc.dirInfos, t),
				System: buildSystem(tc.syncs),
			}

			v := NewSyncResourcesValidator()
			root.Accept(v)

			if tc.shouldFail {
				vettesting.ExpectErrors([]string{vet.UnsyncableNamespaceObjectErrorCode}, v.Error(), t)
			} else {
				vettesting.ExpectErrors([]string{}, v.Error(), t)
			}
		})
	}
}
