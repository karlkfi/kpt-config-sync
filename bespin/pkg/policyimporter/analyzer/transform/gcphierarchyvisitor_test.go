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
	bespinvt "github.com/google/nomos/bespin/pkg/policyimporter/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/node"
	visitorpkg "github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	vt "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
	corev1 "k8s.io/api/core/v1"
)

func TestMultiTopOrgFolderProject(t *testing.T) {
	de := ast.Extension{}
	org1 := bespinvt.Helper.GCPOrg("org1")
	org2 := bespinvt.Helper.GCPOrg("org2")
	org3 := bespinvt.Helper.GCPOrg("org3")
	folder1 := bespinvt.Helper.GCPFolder("folder1")
	folder2 := bespinvt.Helper.GCPFolder("folder2")
	folder3 := bespinvt.Helper.GCPFolder("folder3")
	project1 := bespinvt.Helper.GCPProject("project1")
	project2 := bespinvt.Helper.GCPProject("project2")
	project3 := bespinvt.Helper.GCPProject("project3")

	var tests = []struct {
		name string
		root *ast.Root
		want *ast.Root
	}{
		{
			name: "Single root organization",
			root: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Type:    node.AbstractNamespace,
							Objects: vt.ObjectSets(org1),
						},
					},
				},
			},
			want: &ast.Root{
				Cluster: &ast.Cluster{
					Objects: vt.ClusterObjectSets(org1),
				},
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Type: node.AbstractNamespace,
							Data: de.Add(gcpAttachmentPointKeyType{}, &corev1.ObjectReference{Kind: "Organization", Name: "org1"}),
						},
					},
				},
			},
		},
		{
			name: "Multiple root organizations",
			root: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					Children: []*ast.TreeNode{
						{
							Type:    node.AbstractNamespace,
							Objects: vt.ObjectSets(org1),
						},
						{
							Type:    node.AbstractNamespace,
							Objects: vt.ObjectSets(org2),
						},
						{
							Type:    node.AbstractNamespace,
							Objects: vt.ObjectSets(org3),
						},
					},
				},
			},
			want: &ast.Root{
				Cluster: &ast.Cluster{
					Objects: vt.ClusterObjectSets(org1, org2, org3),
				},
				Tree: &ast.TreeNode{
					Children: []*ast.TreeNode{
						{
							Type: node.AbstractNamespace,
							Data: de.Add(gcpAttachmentPointKeyType{}, &corev1.ObjectReference{Kind: "Organization", Name: "org1"}),
						},
						{
							Type: node.AbstractNamespace,
							Data: de.Add(gcpAttachmentPointKeyType{}, &corev1.ObjectReference{Kind: "Organization", Name: "org2"}),
						},
						{
							Type: node.AbstractNamespace,
							Data: de.Add(gcpAttachmentPointKeyType{}, &corev1.ObjectReference{Kind: "Organization", Name: "org3"}),
						},
					},
				},
			},
		},
		{
			name: "Single root folder",
			root: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Type:    node.AbstractNamespace,
							Objects: vt.ObjectSets(folder1),
						},
					},
				},
			},
			want: &ast.Root{
				Cluster: &ast.Cluster{
					Objects: vt.ClusterObjectSets(folder1),
				},
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Type: node.AbstractNamespace,
							Data: de.Add(gcpAttachmentPointKeyType{}, &corev1.ObjectReference{Kind: "Folder", Name: "folder1"}),
						},
					},
				},
			},
		},
		{
			name: "Multiple root folders",
			root: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Type:    node.AbstractNamespace,
							Objects: vt.ObjectSets(folder1),
						},
						{
							Type:    node.AbstractNamespace,
							Objects: vt.ObjectSets(folder2),
						},
						{
							Type:    node.AbstractNamespace,
							Objects: vt.ObjectSets(folder3),
						},
					},
				},
			},
			want: &ast.Root{
				Cluster: &ast.Cluster{
					Objects: vt.ClusterObjectSets(folder1, folder2, folder3),
				},
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Type: node.AbstractNamespace,
							Data: de.Add(gcpAttachmentPointKeyType{}, &corev1.ObjectReference{Kind: "Folder", Name: "folder1"}),
						},
						{
							Type: node.AbstractNamespace,
							Data: de.Add(gcpAttachmentPointKeyType{}, &corev1.ObjectReference{Kind: "Folder", Name: "folder2"}),
						},
						{
							Type: node.AbstractNamespace,
							Data: de.Add(gcpAttachmentPointKeyType{}, &corev1.ObjectReference{Kind: "Folder", Name: "folder3"}),
						},
					},
				},
			},
		},
		{
			name: "Single root project",
			root: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Type:    node.AbstractNamespace,
							Objects: vt.ObjectSets(project1),
						},
					},
				},
			},
			want: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							// Project tree node is namespace scope.
							Type:    node.Namespace,
							Objects: vt.ObjectSets(project1),
							Data:    de.Add(gcpAttachmentPointKeyType{}, &corev1.ObjectReference{Kind: "Project", Name: "project1"}),
						},
					},
				},
			},
		},
		{
			name: "Multiple root projects",
			root: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Type:    node.AbstractNamespace,
							Objects: vt.ObjectSets(project1),
						},
						{
							Type:    node.AbstractNamespace,
							Objects: vt.ObjectSets(project2),
						},
						{
							Type:    node.AbstractNamespace,
							Objects: vt.ObjectSets(project3),
						},
					},
				},
			},
			want: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Type:    node.Namespace,
							Objects: vt.ObjectSets(project1),
							Data:    de.Add(gcpAttachmentPointKeyType{}, &corev1.ObjectReference{Kind: "Project", Name: "project1"}),
						},
						{
							Type:    node.Namespace,
							Objects: vt.ObjectSets(project2),
							Data:    de.Add(gcpAttachmentPointKeyType{}, &corev1.ObjectReference{Kind: "Project", Name: "project2"}),
						},
						{
							Type:    node.Namespace,
							Objects: vt.ObjectSets(project3),
							Data:    de.Add(gcpAttachmentPointKeyType{}, &corev1.ObjectReference{Kind: "Project", Name: "project3"}),
						},
					},
				},
			},
		},
		{
			name: "Multiple root orgs folders and projects",
			root: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Type:    node.AbstractNamespace,
							Objects: vt.ObjectSets(org1),
						},
						{
							Type:    node.AbstractNamespace,
							Objects: vt.ObjectSets(org2),
						},
						{
							Type:    node.AbstractNamespace,
							Objects: vt.ObjectSets(org3),
						},
						{
							Type:    node.AbstractNamespace,
							Objects: vt.ObjectSets(folder1),
						},
						{
							Type:    node.AbstractNamespace,
							Objects: vt.ObjectSets(folder2),
						},
						{
							Type:    node.AbstractNamespace,
							Objects: vt.ObjectSets(folder3),
						},
						{
							Type:    node.Namespace,
							Objects: vt.ObjectSets(project1),
						},
						{
							Type:    node.Namespace,
							Objects: vt.ObjectSets(project2),
						},
						{
							Type:    node.Namespace,
							Objects: vt.ObjectSets(project3),
						},
					},
				},
			},
			want: &ast.Root{
				Cluster: &ast.Cluster{
					Objects: vt.ClusterObjectSets(org1, org2, org3, folder1, folder2, folder3),
				},
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Type: node.AbstractNamespace,
							Data: de.Add(gcpAttachmentPointKeyType{}, &corev1.ObjectReference{Kind: "Organization", Name: "org1"}),
						},
						{
							Type: node.AbstractNamespace,
							Data: de.Add(gcpAttachmentPointKeyType{}, &corev1.ObjectReference{Kind: "Organization", Name: "org2"}),
						},
						{
							Type: node.AbstractNamespace,
							Data: de.Add(gcpAttachmentPointKeyType{}, &corev1.ObjectReference{Kind: "Organization", Name: "org3"}),
						},
						{
							Type: node.AbstractNamespace,
							Data: de.Add(gcpAttachmentPointKeyType{}, &corev1.ObjectReference{Kind: "Folder", Name: "folder1"}),
						},
						{
							Type: node.AbstractNamespace,
							Data: de.Add(gcpAttachmentPointKeyType{}, &corev1.ObjectReference{Kind: "Folder", Name: "folder2"}),
						},
						{
							Type: node.AbstractNamespace,
							Data: de.Add(gcpAttachmentPointKeyType{}, &corev1.ObjectReference{Kind: "Folder", Name: "folder3"}),
						},
						{
							Type:    node.Namespace,
							Objects: vt.ObjectSets(project1),
							Data:    de.Add(gcpAttachmentPointKeyType{}, &corev1.ObjectReference{Kind: "Project", Name: "project1"}),
						},
						{
							Type:    node.Namespace,
							Objects: vt.ObjectSets(project2),
							Data:    de.Add(gcpAttachmentPointKeyType{}, &corev1.ObjectReference{Kind: "Project", Name: "project2"}),
						},
						{
							Type:    node.Namespace,
							Objects: vt.ObjectSets(project3),
							Data:    de.Add(gcpAttachmentPointKeyType{}, &corev1.ObjectReference{Kind: "Project", Name: "project3"}),
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Make sure the Copying visitor doesn't mutate the tree.
			copier := visitorpkg.NewCopying()
			copier.SetImpl(copier)
			rootCopy := tc.root.Accept(copier)
			verifyInputUnmodified(t, tc.root, rootCopy)

			// Run GCP hierarchy visitor.
			visitor := NewGCPHierarchyVisitor()
			output := tc.root.Accept(visitor)
			if err := visitor.Error(); err != nil {
				t.Errorf("GCP hierarchy visitor resulted in error: %v", err)
			}
			if diff := cmp.Diff(output, tc.want); diff != "" {
				t.Errorf("GCP hierarchy visitor got wrong output.\ngot:\n%+v\nwant:\n%+v\ndiff:\n%s", output, tc.want, diff)
			}
		})
	}
}

func TestFolderAndOrg(t *testing.T) {
	de := ast.Extension{}
	org := bespinvt.Helper.GCPOrg("org-sample")
	folder := bespinvt.Helper.GCPFolder("folder-sample")
	folderUnderOrg := bespinvt.Helper.GCPFolder("folder-under-org-sample")
	folderUnderOrgWithParentRef := folderUnderOrg.DeepCopy()
	folderUnderOrgWithParentRef.Spec.ParentRef = corev1.ObjectReference{
		Kind: org.TypeMeta.Kind,
		Name: org.ObjectMeta.Name,
	}
	subFolder := bespinvt.Helper.GCPFolder("subfolder-sample")
	subFolderWithParentRef := subFolder.DeepCopy()
	subFolderWithParentRef.Spec.ParentRef = corev1.ObjectReference{
		Kind: folder.TypeMeta.Kind,
		Name: folder.ObjectMeta.Name,
	}

	var tests = []struct {
		name  string
		input *ast.Root
		want  *ast.Root
	}{
		// These tests need Cluster initialized or bad things happen.
		{
			name: "Organization should be cluster scoped",
			input: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Type:    node.AbstractNamespace,
							Objects: vt.ObjectSets(org),
						},
					},
				},
			},
			want: &ast.Root{
				Cluster: &ast.Cluster{
					Objects: vt.ClusterObjectSets(org),
				},
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Type: node.AbstractNamespace,
							Data: de.Add(gcpAttachmentPointKeyType{}, &corev1.ObjectReference{Kind: "Organization", Name: "org-sample"}),
						},
					},
				},
			},
		},
		{
			name: "Folder should be cluster scoped",
			input: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Type:    node.AbstractNamespace,
							Objects: vt.ObjectSets(folder),
						},
					},
				},
			},
			want: &ast.Root{
				Cluster: &ast.Cluster{
					Objects: vt.ClusterObjectSets(folder),
				},
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Type: node.AbstractNamespace,
							Data: de.Add(gcpAttachmentPointKeyType{}, &corev1.ObjectReference{Kind: "Folder", Name: "folder-sample"}),
						},
					},
				},
			},
		},
		{
			name: "Folder under an organization should be cluster scoped with parent reference",
			input: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Type:    node.AbstractNamespace,
							Objects: vt.ObjectSets(org),
							Children: []*ast.TreeNode{
								{
									Type:    node.AbstractNamespace,
									Objects: vt.ObjectSets(folderUnderOrg),
								},
							},
						},
					},
				},
			},
			want: &ast.Root{
				Cluster: &ast.Cluster{
					Objects: vt.ClusterObjectSets(folderUnderOrgWithParentRef, org),
				},
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Type: node.AbstractNamespace,
							Data: de.Add(gcpAttachmentPointKeyType{}, &corev1.ObjectReference{Kind: "Organization", Name: "org-sample"}),
							Children: []*ast.TreeNode{
								{
									Type: node.AbstractNamespace,
									Data: de.Add(gcpAttachmentPointKeyType{}, &corev1.ObjectReference{Kind: "Folder", Name: "folder-under-org-sample"}),
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Folder under another folder should be cluster scoped with parent reference",
			input: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Type:    node.AbstractNamespace,
							Objects: vt.ObjectSets(folder),
							Children: []*ast.TreeNode{
								{
									Type:    node.AbstractNamespace,
									Objects: vt.ObjectSets(subFolder),
								},
							},
						},
					},
				},
			},
			want: &ast.Root{
				Cluster: &ast.Cluster{
					Objects: vt.ClusterObjectSets(subFolderWithParentRef, folder),
				},
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Type: node.AbstractNamespace,
							Data: de.Add(gcpAttachmentPointKeyType{}, &corev1.ObjectReference{Kind: "Folder", Name: "folder-sample"}),
							Children: []*ast.TreeNode{
								{
									Type: node.AbstractNamespace,
									Data: de.Add(gcpAttachmentPointKeyType{}, &corev1.ObjectReference{Kind: "Folder", Name: "subfolder-sample"}),
								},
							},
						},
					},
				},
			},
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
			inputCopy := tc.input.Accept(copier)
			visitor := NewGCPHierarchyVisitor()
			output := tc.input.Accept(visitor)
			verifyInputUnmodified(t, tc.input, inputCopy)
			if err := visitor.Error(); err != nil {
				t.Errorf("GCP hierarchy visitor resulted in error: %v", err)
			}
			if diff := cmp.Diff(output, tc.want); diff != "" {
				t.Errorf("got diff:\n%v", diff)
			}
		})
	}
}

func TestProject(t *testing.T) {
	de := ast.Extension{}
	org := bespinvt.Helper.GCPOrg("org")
	folder := bespinvt.Helper.GCPFolder("folder")
	project := bespinvt.Helper.GCPProject("project")
	projectUnderOrgWithParentRef := project.DeepCopy()
	projectUnderOrgWithParentRef.Spec.ParentRef = corev1.ObjectReference{
		Kind: org.TypeMeta.Kind,
		Name: org.ObjectMeta.Name,
	}
	projectUnderFolderWithParentRef := project.DeepCopy()
	projectUnderFolderWithParentRef.Spec.ParentRef = corev1.ObjectReference{
		Kind: folder.TypeMeta.Kind,
		Name: folder.ObjectMeta.Name,
	}

	var tests = []struct {
		name  string
		input *ast.Root
		want  *ast.Root
	}{
		{
			name: "Project under organization should be namespace scoped with parent reference",
			input: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Type:    node.AbstractNamespace,
							Objects: vt.ObjectSets(org),
							Children: []*ast.TreeNode{
								{
									Type:    node.AbstractNamespace,
									Objects: vt.ObjectSets(project),
								},
							},
						},
					},
				},
			},
			want: &ast.Root{
				Cluster: &ast.Cluster{
					Objects: vt.ClusterObjectSets(org),
				},
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Type: node.AbstractNamespace,
							Children: []*ast.TreeNode{
								{
									Type:    node.Namespace,
									Objects: vt.ObjectSets(projectUnderOrgWithParentRef),
									Data:    de.Add(gcpAttachmentPointKeyType{}, &corev1.ObjectReference{Kind: "Project", Name: "project"}),
								},
							},
							Data: de.Add(gcpAttachmentPointKeyType{}, &corev1.ObjectReference{Kind: "Organization", Name: "org"}),
						},
					},
				},
			},
		},
		{
			name: "Project under folder should be namespace scoped with parent reference",
			input: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Type:    node.AbstractNamespace,
							Objects: vt.ObjectSets(folder),
							Children: []*ast.TreeNode{
								{
									Type:    node.AbstractNamespace,
									Objects: vt.ObjectSets(project),
								},
							},
						},
					},
				},
			},
			want: &ast.Root{
				Cluster: &ast.Cluster{
					Objects: vt.ClusterObjectSets(folder),
				},
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Type: node.AbstractNamespace,
							Children: []*ast.TreeNode{
								{
									Type:    node.Namespace,
									Objects: vt.ObjectSets(projectUnderFolderWithParentRef),
									Data:    de.Add(gcpAttachmentPointKeyType{}, &corev1.ObjectReference{Kind: "Project", Name: "project"}),
								},
							},
							Data: de.Add(gcpAttachmentPointKeyType{}, &corev1.ObjectReference{Kind: "Folder", Name: "folder"}),
						},
					},
				},
			},
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
			inputCopy := tc.input.Accept(copier)
			visitor := NewGCPHierarchyVisitor()
			output := tc.input.Accept(visitor)
			verifyInputUnmodified(t, tc.input, inputCopy)
			if output.Tree == nil || len(output.Tree.Children[0].Children) != 1 {
				t.Fatalf("unexpected output root: %+v", output)
			}
			if diff := cmp.Diff(output, tc.want); diff != "" {
				t.Errorf("got diff:\n%v", diff)
			}
		})
	}
}

func TestAttachmentPoint(t *testing.T) {
	org := bespinvt.Helper.GCPOrg("org")
	folder := bespinvt.Helper.GCPFolder("folder")
	project := bespinvt.Helper.GCPProject("project")

	input := &ast.Root{
		Cluster: &ast.Cluster{},
		Tree: &ast.TreeNode{
			Children: []*ast.TreeNode{
				{
					Objects: vt.ObjectSets(org),
					Children: []*ast.TreeNode{
						{
							Type:    node.AbstractNamespace,
							Objects: vt.ObjectSets(folder),
							Children: []*ast.TreeNode{
								{
									Type:    node.AbstractNamespace,
									Objects: vt.ObjectSets(project),
								},
							},
						},
					},
				},
			},
		},
	}
	visitor := NewGCPHierarchyVisitor()
	output := input.Accept(visitor)
	wantOrgRef := &corev1.ObjectReference{
		Kind: org.TypeMeta.Kind,
		Name: org.ObjectMeta.Name,
	}
	orgNode := output.Tree.Children[0]
	verifyAttachmentPoint(t, orgNode, wantOrgRef)

	wantFolderRef := &corev1.ObjectReference{
		Kind: folder.TypeMeta.Kind,
		Name: folder.ObjectMeta.Name,
	}
	folderNode := orgNode.Children[0]
	verifyAttachmentPoint(t, folderNode, wantFolderRef)

	wantProjectRef := &corev1.ObjectReference{
		Kind: project.TypeMeta.Kind,
		Name: project.ObjectMeta.Name,
	}
	projectNode := folderNode.Children[0]
	verifyAttachmentPoint(t, projectNode, wantProjectRef)
}

func verifyAttachmentPoint(t *testing.T, node *ast.TreeNode, wantRef *corev1.ObjectReference) {
	gotRef := node.Data.Get(gcpAttachmentPointKey)
	if !cmp.Equal(gotRef, wantRef) {
		t.Errorf("Got policy attachment point %v, want %v", gotRef, wantRef)
	}
}

func TestHierarchyError(t *testing.T) {
	project := bespinvt.Helper.GCPProject("project")
	project2 := bespinvt.Helper.GCPProject("project2")
	folder := bespinvt.Helper.GCPFolder("folder")
	folder2 := bespinvt.Helper.GCPFolder("folder2")
	org := bespinvt.Helper.GCPFolder("org")
	org2 := bespinvt.Helper.GCPOrg("org2")
	var tests = []struct {
		name  string
		input *ast.Root
	}{
		{
			name: "GCP Resources defined at top directory",
			input: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					// Error: resources shouldn't be defined at top directory.
					Objects: vt.ObjectSets(org),
					Children: []*ast.TreeNode{
						{
							Type:    node.AbstractNamespace,
							Objects: vt.ObjectSets(folder),
							Children: []*ast.TreeNode{
								{
									Type: node.Namespace,
									// Error: project with Project parent.
									Objects: vt.ObjectSets(project2),
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Multiple orgs at same tree node",
			input: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					Children: []*ast.TreeNode{
						{
							Type: node.AbstractNamespace,
							// Error: multiple orgs at same tree node.
							Objects: vt.ObjectSets(org, org2),
						},
					},
				},
			},
		},
		{
			name: "Multiple folders at same tree node",
			input: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					Children: []*ast.TreeNode{
						{
							Type: node.AbstractNamespace,
							// Error: multiple folders at same tree node.
							Objects: vt.ObjectSets(folder, folder2),
						},
					},
				},
			},
		},
		{
			name: "Multiple projets at same tree node",
			input: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					Children: []*ast.TreeNode{
						{
							Type: node.Namespace,
							// Error: multiple projects at same tree node.
							Objects: vt.ObjectSets(project, project2),
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
					Children: []*ast.TreeNode{
						{
							Type: node.Namespace,
							// Parent project.
							Objects: vt.ObjectSets(project),
							Children: []*ast.TreeNode{
								{
									Type: node.Namespace,
									// Error: project with Project parent.
									Objects: vt.ObjectSets(project2),
								},
							},
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
					Children: []*ast.TreeNode{
						{
							Type:    node.Namespace,
							Objects: vt.ObjectSets(project),
							Children: []*ast.TreeNode{
								{
									Type:    node.AbstractNamespace,
									Objects: vt.ObjectSets(folder),
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Org with org (any) parent",
			input: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					Children: []*ast.TreeNode{
						{
							Objects: vt.ObjectSets(org),
							Children: []*ast.TreeNode{
								{
									Children: []*ast.TreeNode{
										{
											Type:    node.AbstractNamespace,
											Objects: vt.ObjectSets(org2),
										},
									},
								},
							},
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
					Children: []*ast.TreeNode{
						{
							Objects: vt.ObjectSets(org),
							Children: []*ast.TreeNode{
								{
									Type:    node.AbstractNamespace,
									Objects: vt.ObjectSets(), // No objects exist meaning this level of directory is empty.
									Children: []*ast.TreeNode{
										{
											Type:    node.AbstractNamespace,
											Objects: vt.ObjectSets(folder),
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Root tree node without an organization/folder/project resource",
			input: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					Children: []*ast.TreeNode{
						{
							Type: node.AbstractNamespace,
							// Error: tree node with no organization/folder/project resource defined.
							Objects: vt.ObjectSets(),
						},
					},
				},
			},
		},
		{
			name: "Child tree node without folder/project resource",
			input: &ast.Root{
				Cluster: &ast.Cluster{},
				Tree: &ast.TreeNode{
					Children: []*ast.TreeNode{
						{
							Type:    node.AbstractNamespace,
							Objects: vt.ObjectSets(folder),
							Children: []*ast.TreeNode{
								{
									Type: node.AbstractNamespace,
									// Error: tree node with no folder/project resource defined.
									Objects: vt.ObjectSets(),
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			visitor := NewGCPHierarchyVisitor()
			_ = tc.input.Accept(visitor)
			if err := visitor.Error(); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func verifyInputUnmodified(t *testing.T, input, inputCopy *ast.Root) {
	// Mutation indicates something was implemented wrong, the input shouldn't be modified.
	if diff := cmp.Diff(input, inputCopy); diff != "" {
		t.Errorf("input mutated while running visitor: %s", diff)
	}
}
