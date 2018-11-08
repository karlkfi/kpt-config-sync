/*
Copyright 2018 The Nomos Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package transform

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/api/policyascode/v1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	visitorpkg "github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	vt "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestFolderAndOrg(t *testing.T) {
	org := vt.Helper.GCPOrg()
	folder := vt.Helper.GCPFolder()
	folderUnderOrg := vt.Helper.GCPFolder()
	folderUnderOrg.Spec.ParentReference = v1.ParentReference{
		Kind: org.TypeMeta.Kind,
		Name: org.ObjectMeta.Name,
	}
	subFolder := vt.Helper.GCPFolder()
	subFolder.ObjectMeta.Name = "subfolder-sample"
	subFolderWithParentRef := subFolder.DeepCopy()
	subFolderWithParentRef.Spec.ParentReference = v1.ParentReference{
		Kind: folder.TypeMeta.Kind,
		Name: folder.ObjectMeta.Name,
	}

	var tests = []struct {
		name  string
		input *ast.Root
		want  ast.ClusterObjectList
	}{
		// These tests need Cluster initialized or bad things happen.
		{
			name: "Organization should be cluster scoped",
			input: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					Objects: vt.ObjectSets(org),
				},
			},
			want: vt.ClusterObjectSets(org),
		},
		{
			name: "Folder should be cluster scoped",
			input: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					Objects: vt.ObjectSets(folder),
				},
			},
			want: vt.ClusterObjectSets(folder),
		},
		{
			name: "Folder under an organization should be cluster scoped with parent reference",
			input: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					Objects: vt.ObjectSets(org),
					Children: []*ast.TreeNode{
						&ast.TreeNode{
							Type:    ast.AbstractNamespace,
							Objects: vt.ObjectSets(folder),
						},
					},
				},
			},
			want: vt.ClusterObjectSets(folderUnderOrg, org),
		},
		{
			name: "Folder under another folder should be cluster scoped with parent reference",
			input: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					Objects: vt.ObjectSets(folder),
					Children: []*ast.TreeNode{
						&ast.TreeNode{
							Type:    ast.AbstractNamespace,
							Objects: vt.ObjectSets(subFolder),
						},
					},
				},
			},
			want: vt.ClusterObjectSets(subFolderWithParentRef, folder),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// This was copied from MutatingVisitorTest and then modified.
			// This removes much of the implementation detail and Nomos-specific
			// code, which makes it a bit clearer what is going on for Bespin's
			// needs.
			copier := visitorpkg.NewCopying()
			copier.SetImpl(copier)
			inputCopy, ok := tc.input.Accept(copier).(*ast.Root)
			if !ok {
				t.Fatalf(
					"framework error: return value from copying visitor needs to be of type *ast.Root, got: %#v", inputCopy)
			}
			visitor := NewGCPHierarchyVisitor()
			output := tc.input.Accept(visitor).(*ast.Root)
			verifyInputUnmodified(t, tc.input, inputCopy)
			if err := visitor.Error(); err != nil {
				t.Errorf("GCP hierarchy visitor resulted in error: %v", err)
			}
			if diff := cmp.Diff(tc.want, output.Cluster.Objects); diff != "" {
				t.Errorf("got diff:\n%v", diff)
			}
		})
	}
}

func TestProject(t *testing.T) {
	org := vt.Helper.GCPOrg()
	folder := vt.Helper.GCPFolder()
	project := vt.Helper.GCPProject()
	projectUnderOrg := vt.Helper.GCPProject()
	projectUnderOrg.Spec.ParentReference = v1.ParentReference{
		Kind: org.TypeMeta.Kind,
		Name: org.ObjectMeta.Name,
	}
	projectUnderFolder := vt.Helper.GCPProject()
	projectUnderFolder.Spec.ParentReference = v1.ParentReference{
		Kind: folder.TypeMeta.Kind,
		Name: folder.ObjectMeta.Name,
	}

	var tests = []struct {
		name  string
		input *ast.Root
		want  ast.ObjectList
	}{
		{
			name: "Project under organization should be namespace scoped with parent reference",
			input: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					Objects: vt.ObjectSets(org),
					Children: []*ast.TreeNode{
						&ast.TreeNode{
							Type:    ast.AbstractNamespace,
							Objects: vt.ObjectSets(project),
						},
					},
				},
			},
			want: vt.ObjectSets(projectUnderOrg),
		},
		{
			name: "Project under folder should be namespace scoped with parent reference",
			input: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					Objects: vt.ObjectSets(folder),
					Children: []*ast.TreeNode{
						&ast.TreeNode{
							Type:    ast.AbstractNamespace,
							Objects: vt.ObjectSets(project),
						},
					},
				},
			},
			want: vt.ObjectSets(projectUnderFolder),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// This was copied from MutatingVisitorTest and then modified.
			// This removes much of the implementation detail and Nomos-specific
			// code, which makes it a bit clearer what is going on for Bespin's
			// needs.
			copier := visitorpkg.NewCopying()
			copier.SetImpl(copier)
			inputCopy, ok := tc.input.Accept(copier).(*ast.Root)
			if !ok {
				t.Fatalf(
					"framework error: return value from copying visitor needs to be of type *ast.Root, got: %#v", inputCopy)
			}
			visitor := NewGCPHierarchyVisitor()
			output := tc.input.Accept(visitor).(*ast.Root)
			verifyInputUnmodified(t, tc.input, inputCopy)
			if output.Tree == nil || len(output.Tree.Children) != 1 {
				t.Fatalf("unexpected output root: %+v", output)
			}
			projectNode := output.Tree.Children[0]
			if diff := cmp.Diff(tc.want, projectNode.Objects); diff != "" {
				t.Errorf("got diff:\n%v", diff)
			}
		})
	}
}

func TestHierarchyError(t *testing.T) {
	project := vt.Helper.GCPProject()
	project2 := vt.Helper.GCPProject()
	project2.ObjectMeta.Name = "project2"
	folder := vt.Helper.GCPFolder()
	org := vt.Helper.GCPFolder()
	var tests = []struct {
		name  string
		input *ast.Root
	}{
		{
			name: "Project w/o parent",
			input: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					Objects: vt.ObjectSets(project),
				},
			},
		},
		{
			name: "Project w/o a GCP folder/org parent",
			input: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					Children: []*ast.TreeNode{
						&ast.TreeNode{
							Type:    ast.AbstractNamespace,
							Objects: vt.ObjectSets(project),
						},
					},
				},
			},
		},
		{
			name: "Project with project parent",
			input: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					Objects: vt.ObjectSets(project),
					Children: []*ast.TreeNode{
						&ast.TreeNode{
							Type:    ast.AbstractNamespace,
							Objects: vt.ObjectSets(project2),
						},
					},
				},
			},
		},
		{
			name: "Folder with project parent",
			input: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					Objects: vt.ObjectSets(project),
					Children: []*ast.TreeNode{
						&ast.TreeNode{
							Type:    ast.AbstractNamespace,
							Objects: vt.ObjectSets(folder),
						},
					},
				},
			},
		},
		{
			name: "Org with parent",
			input: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					Children: []*ast.TreeNode{
						&ast.TreeNode{
							Type:    ast.AbstractNamespace,
							Objects: vt.ObjectSets(org),
						},
					},
				},
			},
		},
		{
			name: "Gap in hierarchy",
			input: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					Objects: vt.ObjectSets(org),
					Children: []*ast.TreeNode{
						&ast.TreeNode{
							Type: ast.AbstractNamespace,
							Children: []*ast.TreeNode{
								&ast.TreeNode{
									Type:    ast.AbstractNamespace,
									Objects: vt.ObjectSets(folder),
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Cannot have both org and folder at the same level",
			input: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					Type:    ast.AbstractNamespace,
					Objects: vt.ObjectSets(org, folder),
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			visitor := NewGCPHierarchyVisitor()
			_ = tc.input.Accept(visitor).(*ast.Root)
			if err := visitor.Error(); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func verifyInputUnmodified(t *testing.T, input, inputCopy *ast.Root) {
	versionCmp := cmp.Comparer(func(lhs, rhs resource.Quantity) bool {
		return lhs.Cmp(rhs) == 0
	})
	// Mutation indicates something was implemented wrong, the input shouldn't be modified.
	if diff := cmp.Diff(input, inputCopy, versionCmp); diff != "" {
		t.Errorf("input mutated while running visitor: %s", diff)
	}
}
