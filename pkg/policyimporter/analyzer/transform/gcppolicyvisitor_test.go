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
	"k8s.io/apimachinery/pkg/runtime"
)

func TestIAMPolicies(t *testing.T) {
	project := vt.Helper.GCPProject()

	var tests = []struct {
		name   string
		policy *v1.IAMPolicy
		want   v1.ResourceReference
	}{
		{
			name: "IAM policies should gain a project attachment point",
			policy: &v1.IAMPolicy{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1.SchemeGroupVersion.String(),
					Kind:       v1.IAMPolicyKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "iam-policy",
				},
				Spec: v1.IAMPolicySpec{
					Bindings: []v1.IAMPolicyBinding{},
				},
			},
			want: v1.ResourceReference{
				Kind: project.TypeMeta.Kind,
				Name: project.ObjectMeta.Name,
			},
		},
		// TODO(cflewis): Fill these in later.
		{
			name: "An IAM policy without an attachment point should fail",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.policy == nil {
				t.Skip("test is stubbed")
			}
			runAttachmentPointTest(t, project, tc.policy, tc.want)
		})
	}
}

func TestOrgPolicies(t *testing.T) {
	project := vt.Helper.GCPProject()

	var tests = []struct {
		name   string
		policy *v1.OrganizationPolicy
		want   v1.ResourceReference
	}{
		{
			name: "Organization policies should gain a project attachment point",
			policy: &v1.OrganizationPolicy{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1.SchemeGroupVersion.String(),
					Kind:       v1.OrganizationPolicyKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "org-policy",
				},
				Spec: v1.OrganizationPolicySpec{
					Constraints: []v1.OrganizationPolicyConstraint{},
				},
			},
			want: v1.ResourceReference{
				Kind: project.TypeMeta.Kind,
				Name: project.ObjectMeta.Name,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runAttachmentPointTest(t, project, tc.policy, tc.want)
		})
	}
}

func runAttachmentPointTest(t *testing.T, project *v1.Project, policy runtime.Object, want v1.ResourceReference) {
	input := &ast.Root{
		Cluster: &ast.Cluster{},
		Tree: &ast.TreeNode{
			Objects: vt.ObjectSets(vt.Helper.GCPOrg()),
			Children: []*ast.TreeNode{
				&ast.TreeNode{
					Type:    ast.AbstractNamespace,
					Objects: vt.ObjectSets(project, policy),
				},
			},
		},
	}

	input.Tree.Data = input.Tree.Data.Add(gcpAttachmentPointKey, nil)
	projectNode := input.Tree.Children[0]
	projectNode.Data = projectNode.Data.Add(gcpAttachmentPointKey, &want)

	copier := visitorpkg.NewCopying()
	copier.SetImpl(copier)
	inputCopy := input.Accept(copier)

	visitor := NewGCPPolicyVisitor()
	output := input.Accept(visitor)
	verifyInputUnmodified(t, input, inputCopy)
	if err := visitor.Error(); err != nil {
		t.Errorf("GCP hierarchy visitor resulted in error: %v", err)
	}

	if output.Tree == nil || len(output.Tree.Children) != 1 {
		t.Fatalf("unexpected output root: %+v", output)
	}

	wantObj := policy.DeepCopyObject()

	// It's impossible to collapse these two type cases as Go won't convert v to a concrete type because
	// it doesn't know which to pick. This means the code has to be repeated for each type case.
	switch v := wantObj.(type) {
	case *v1.IAMPolicy:
		v.Spec.ResourceReference = want
	case *v1.OrganizationPolicy:
		v.Spec.ResourceReference = want
	default:
		t.Fatal("unknown policy type")
	}

	projectNode = output.Tree.Children[0]
	if diff := cmp.Diff(vt.ObjectSets(project, wantObj), projectNode.Objects); diff != "" {
		t.Errorf("got diff:\n%v", diff)
	}
}

func TestIAMPolicyConversion(t *testing.T) {
	org := vt.Helper.GCPOrg()
	project := vt.Helper.GCPProject()

	var tests = []struct {
		name   string
		policy *v1.IAMPolicy
		want   *v1.ClusterIAMPolicy
	}{
		{
			name: "A non-project attachment point should attach to a cluster instead",
			policy: &v1.IAMPolicy{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1.SchemeGroupVersion.String(),
					Kind:       v1.IAMPolicyKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "iam-policy",
				},
				Spec: v1.IAMPolicySpec{
					Bindings: []v1.IAMPolicyBinding{},
				},
			},
			want: &v1.ClusterIAMPolicy{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1.SchemeGroupVersion.String(),
					Kind:       v1.ClusterIAMPolicyKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "iam-policy",
				},
				Spec: v1.IAMPolicySpec{
					Bindings: []v1.IAMPolicyBinding{},
					ResourceReference: v1.ResourceReference{
						Kind: org.TypeMeta.Kind,
						Name: org.ObjectMeta.Name,
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runClusterObjectsTest(t, org, project, tc.policy, tc.want)
		})
	}
}

func TestOrgPolicyConversion(t *testing.T) {
	org := vt.Helper.GCPOrg()
	project := vt.Helper.GCPProject()

	var tests = []struct {
		name   string
		policy *v1.OrganizationPolicy
		want   *v1.ClusterOrganizationPolicy
	}{
		{
			name: "A non-project attachment point should attach to a cluster instead",
			policy: &v1.OrganizationPolicy{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1.SchemeGroupVersion.String(),
					Kind:       v1.OrganizationPolicyKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "org-policy",
				},
				Spec: v1.OrganizationPolicySpec{
					ResourceReference: v1.ResourceReference{
						Kind: org.TypeMeta.Kind,
						Name: org.ObjectMeta.Name,
					},
				},
			},
			want: &v1.ClusterOrganizationPolicy{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1.SchemeGroupVersion.String(),
					Kind:       v1.ClusterOrganizationPolicyKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "org-policy",
				},
				Spec: v1.OrganizationPolicySpec{
					ResourceReference: v1.ResourceReference{
						Kind: org.TypeMeta.Kind,
						Name: org.ObjectMeta.Name,
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runClusterObjectsTest(t, org, project, tc.policy, tc.want)
		})
	}
}

func runClusterObjectsTest(t *testing.T, org *v1.Organization, project *v1.Project, policy, want runtime.Object) {
	input := &ast.Root{
		Cluster: &ast.Cluster{},
		Tree: &ast.TreeNode{
			Objects: vt.ObjectSets(vt.Helper.GCPOrg()),
			Children: []*ast.TreeNode{
				&ast.TreeNode{
					Type:    ast.AbstractNamespace,
					Objects: vt.ObjectSets(project, policy),
				},
			},
		},
	}

	input.Tree.Data = input.Tree.Data.Add(gcpAttachmentPointKey, nil)
	projectNode := input.Tree.Children[0]

	switch v := want.(type) {
	case *v1.ClusterIAMPolicy:
		projectNode.Data = projectNode.Data.Add(gcpAttachmentPointKey, &v.Spec.ResourceReference)
	case *v1.ClusterOrganizationPolicy:
		projectNode.Data = projectNode.Data.Add(gcpAttachmentPointKey, &v.Spec.ResourceReference)
	}

	copier := visitorpkg.NewCopying()
	copier.SetImpl(copier)
	inputCopy := input.Accept(copier)
	visitor := NewGCPPolicyVisitor()
	output := input.Accept(visitor)
	verifyInputUnmodified(t, input, inputCopy)
	if err := visitor.Error(); err != nil {
		t.Errorf("GCP hierarchy visitor resulted in error: %v", err)
	}

	if output.Tree == nil || len(output.Tree.Children) != 1 {
		t.Fatalf("unexpected output root: %+v", output)
	}

	if diff := cmp.Diff(vt.ClusterObjectSets(want), output.Cluster.Objects); diff != "" {
		t.Errorf("got diff:\n%v", diff)
	}
}
