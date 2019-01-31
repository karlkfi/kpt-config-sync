/*
Copyright 2019 The Nomos Authors.

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
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/bespin/pkg/controllers/resource"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/node"
	visitorpkg "github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	vt "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func addImportAnnotationsFromRoot(r *ast.Root, o runtime.Object) {
	m := o.(metav1.Object)
	a := m.GetAnnotations()
	if a == nil {
		a = make(map[string]string)
	}
	a[resource.ImportTokenKey] = r.ImportToken
	a[resource.ImportTimeKey] = r.LoadTime.Format(time.RFC3339)
	m.SetAnnotations(a)
}

func addImportAnnotations(o runtime.Object) {
	m := o.(metav1.Object)
	a := m.GetAnnotations()
	if a == nil {
		a = make(map[string]string)
	}
	a[resource.ImportTokenKey] = "somethingfaked"
	a[resource.ImportTimeKey] = time.Now().Format(time.RFC3339)
	m.SetAnnotations(a)
}

func addOtherAnnotations(o runtime.Object) {
	m := o.(metav1.Object)
	a := m.GetAnnotations()
	if a == nil {
		a = make(map[string]string)
	}
	a["color"] = "blue"
	m.SetAnnotations(a)
}

func TestApplyAnnotationsWithHierarchy(t *testing.T) {
	r := &ast.Root{
		ImportToken: "6c81e33a81e757e07ceb2fb06e4da1b22f5ef65b",
		LoadTime:    time.Now(),
	}
	org1 := vt.Helper.GCPOrg("org1")
	org1WithAnnotations := org1.DeepCopy()
	addImportAnnotationsFromRoot(r, org1WithAnnotations)

	orgpolicy1 := vt.Helper.GCPOrgPolicy("orgpolicy1")
	addOtherAnnotations(orgpolicy1)
	orgpolicy1WithAnnotations := orgpolicy1.DeepCopy()
	addImportAnnotationsFromRoot(r, orgpolicy1WithAnnotations)

	folder1 := vt.Helper.GCPFolder("folder1")
	addImportAnnotations(folder1)
	folder1WithAnnotations := folder1.DeepCopy()
	addImportAnnotationsFromRoot(r, folder1WithAnnotations)

	fiam1 := vt.Helper.GCPIAMPolicy("fiam1")
	addImportAnnotations(fiam1)
	addOtherAnnotations(fiam1)
	fiam1WithAnnotations := fiam1.DeepCopy()
	addImportAnnotationsFromRoot(r, fiam1WithAnnotations)

	folder2 := vt.Helper.GCPFolder("folder2")
	folder2WithAnnotations := folder2.DeepCopy()
	addImportAnnotationsFromRoot(r, folder2WithAnnotations)

	project1 := vt.Helper.GCPProject("project1")
	project1WithAnnotations := project1.DeepCopy()
	addImportAnnotationsFromRoot(r, project1WithAnnotations)
	piam1 := vt.Helper.GCPIAMPolicy("piam1")
	piam1WithAnnotations := piam1.DeepCopy()
	addImportAnnotationsFromRoot(r, piam1WithAnnotations)

	project2 := vt.Helper.GCPProject("project2")
	addOtherAnnotations(project2)
	project2WithAnnotations := project2.DeepCopy()
	addImportAnnotationsFromRoot(r, project2WithAnnotations)
	piam2 := vt.Helper.GCPIAMPolicy("piam2")
	piam2WithAnnotations := piam2.DeepCopy()
	addImportAnnotationsFromRoot(r, piam2WithAnnotations)

	project3 := vt.Helper.GCPProject("project3")
	addOtherAnnotations(project3)
	project3WithAnnotations := project3.DeepCopy()
	addImportAnnotationsFromRoot(r, project3WithAnnotations)
	piam3 := vt.Helper.GCPIAMPolicy("piam3")
	piam3WithAnnotations := piam3.DeepCopy()
	addImportAnnotationsFromRoot(r, piam3WithAnnotations)

	var tests = []struct {
		name string
		root *ast.Root
		want *ast.Root
	}{
		{
			name: "All resources along the hierarchy should have import token and import time annotated",
			root: &ast.Root{
				ImportToken: r.ImportToken,
				LoadTime:    r.LoadTime,
				Cluster: &ast.Cluster{
					Objects: vt.ClusterObjectSets(org1, orgpolicy1, folder1, fiam1, folder2),
				},
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							// org1
							Type: node.AbstractNamespace,
							Children: []*ast.TreeNode{
								{
									// folder1
									Type: node.AbstractNamespace,
									Children: []*ast.TreeNode{
										{
											// project1
											Type:    node.Namespace,
											Objects: vt.ObjectSets(project1, piam1),
										},
									},
								},
								{
									// folder2
									Type: node.AbstractNamespace,
									Children: []*ast.TreeNode{
										{
											// project2 & project3
											Type:    node.Namespace,
											Objects: vt.ObjectSets(project2, piam2, project3, piam3),
										},
									},
								},
							},
						},
					},
				},
			},
			want: &ast.Root{
				ImportToken: r.ImportToken,
				LoadTime:    r.LoadTime,
				Cluster: &ast.Cluster{
					Objects: vt.ClusterObjectSets(
						org1WithAnnotations, orgpolicy1WithAnnotations, folder1WithAnnotations,
						fiam1WithAnnotations, folder2WithAnnotations),
				},
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							// org1
							Type: node.AbstractNamespace,
							Children: []*ast.TreeNode{
								{
									// folder1
									Type: node.AbstractNamespace,
									Children: []*ast.TreeNode{
										{
											// project1
											Type:    node.Namespace,
											Objects: vt.ObjectSets(project1WithAnnotations, piam1WithAnnotations),
										},
									},
								},
								{
									// folder2
									Type: node.AbstractNamespace,
									Children: []*ast.TreeNode{
										{
											// project2 & project3
											Type: node.Namespace,
											Objects: vt.ObjectSets(
												project2WithAnnotations, piam2WithAnnotations, project3WithAnnotations,
												piam3WithAnnotations),
										},
									},
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
			// Make sure the Copying visitor doesn't mutate the tree.
			copier := visitorpkg.NewCopying()
			copier.SetImpl(copier)
			rootCopy := tc.root.Accept(copier)
			verifyInputUnmodified(t, tc.root, rootCopy)

			visitor := NewGCPAnnotationVisitor()
			output := tc.root.Accept(visitor)
			if diff := cmp.Diff(output, tc.want); diff != "" {
				t.Errorf("GCP annotation visitor got wrong output.\ngot:\n%+v\nwant:\n%+v\ndiff:\n%s", output, tc.want, diff)
			}
		})
	}
}
