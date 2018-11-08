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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestProjectPolicy(t *testing.T) {
	project := vt.Helper.GCPProject()
	iamPolicy := &v1.IAMPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       "IAMPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "iam-policy",
		},
		Spec: v1.IAMPolicySpec{
			Bindings: []v1.IAMPolicyBinding{},
		},
	}
	wantIAMPolicy := iamPolicy.DeepCopy()
	wantIAMPolicy.Spec.ResourceReference = v1.ResourceReference{
		Kind: project.TypeMeta.Kind,
		Name: project.ObjectMeta.Name,
	}
	orgPolicy := &v1.OrganizationPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       "OrganizationPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "org-policy",
		},
		Spec: v1.OrganizationPolicySpec{
			Constraints: []v1.OrganizationPolicyConstraint{},
		},
	}
	wantOrgPolicy := orgPolicy.DeepCopy()
	wantOrgPolicy.Spec.ResourceReference = wantIAMPolicy.Spec.ResourceReference
	input := &ast.Root{
		Cluster: &ast.Cluster{},
		Tree: &ast.TreeNode{
			Objects: vt.ObjectSets(vt.Helper.GCPOrg()),
			Children: []*ast.TreeNode{
				&ast.TreeNode{
					Type:    ast.AbstractNamespace,
					Objects: vt.ObjectSets(project, iamPolicy, orgPolicy),
				},
			},
		},
	}
	input.Tree.Data = input.Tree.Data.Add(gcpAttachmentPointKey, nil)
	projectNode := input.Tree.Children[0]
	projectNode.Data = projectNode.Data.Add(gcpAttachmentPointKey, &wantIAMPolicy.Spec.ResourceReference)

	copier := visitorpkg.NewCopying()
	copier.SetImpl(copier)
	inputCopy, ok := input.Accept(copier).(*ast.Root)
	if !ok {
		t.Fatalf(
			"framework error: return value from copying visitor needs to be of type *ast.Root, got: %#v", inputCopy)
	}
	visitor := NewGCPPolicyVisitor()
	output := input.Accept(visitor).(*ast.Root)
	verifyInputUnmodified(t, input, inputCopy)
	if err := visitor.Error(); err != nil {
		t.Errorf("GCP hierarchy visitor resulted in error: %v", err)
	}
	if output.Tree == nil || len(output.Tree.Children) != 1 {
		t.Fatalf("unexpected output root: %+v", output)
	}
	projectNode = output.Tree.Children[0]
	if diff := cmp.Diff(vt.ObjectSets(project, wantIAMPolicy, wantOrgPolicy), projectNode.Objects); diff != "" {
		t.Errorf("got diff:\n%v", diff)
	}
}
