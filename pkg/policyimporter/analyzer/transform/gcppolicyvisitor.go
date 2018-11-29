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
	"fmt"

	"github.com/google/nomos/pkg/api/policyascode/v1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// GCPPolicyVisitor is a visitor that handles GCP IAM and organization policies.
// Specifically, it:
// * Sets up the reference from GCP policy to their attachment points.
// * Moves certain policies from namespace scope to cluster scope.
// Precondition: GCPHierachyVisitor must run before GCPPolicyVisitor, or else
// the required policy attachment points will be missing.
type GCPPolicyVisitor struct {
	*visitor.Copying
	// For adding cluster scoped policies.
	cluster *ast.Cluster
	// Denotes the current TreeNode while visiting object list.
	currentTreeNode *ast.TreeNode
}

var _ ast.Visitor = &GCPPolicyVisitor{}

// NewGCPPolicyVisitor makes a new visitor.
func NewGCPPolicyVisitor() *GCPPolicyVisitor {
	v := &GCPPolicyVisitor{Copying: visitor.NewCopying()}
	v.SetImpl(v)
	return v
}

// Error implements Visitor.
func (v *GCPPolicyVisitor) Error() error {
	return nil
}

// VisitCluster implements Visitor.
func (v *GCPPolicyVisitor) VisitCluster(c *ast.Cluster) *ast.Cluster {
	newC := v.Copying.VisitCluster(c)
	v.cluster = newC
	return newC
}

// VisitReservedNamespaces implements Visitor. Currently unused and always returns
// the passed node.
func (v *GCPPolicyVisitor) VisitReservedNamespaces(r *ast.ReservedNamespaces) *ast.ReservedNamespaces {
	return r
}

// VisitTreeNode implements Visitor.
func (v *GCPPolicyVisitor) VisitTreeNode(n *ast.TreeNode) *ast.TreeNode {
	v.currentTreeNode = n
	return v.Copying.VisitTreeNode(n)
}

// VisitObject implements Visitor. The precondition is that the poicy attachment
// point has already been set up. It fills in the resource reference in the
// policy spec. If the policy attachment point is cluster scoped (org or folder),
// this method will transform the policy to the cluster scoped version and move
// them over to the list of cluster objects.
func (v *GCPPolicyVisitor) VisitObject(o *ast.NamespaceObject) *ast.NamespaceObject {
	gvk := o.GetObjectKind().GroupVersionKind()
	if gvk.Group != v1.SchemeGroupVersion.Group {
		return o
	}
	var attachmentPoint *v1.ResourceReference
	if ap := v.currentTreeNode.Data.Get(gcpAttachmentPointKey); ap != nil {
		attachmentPoint = ap.(*v1.ResourceReference)
	}
	switch gcpObj := o.FileObject.Object.(type) {
	case *v1.IAMPolicy:
		if attachmentPoint == nil {
			panic(fmt.Sprintf("Missing attachment point for IAM policy %v", o))
		}
		iamPolicy := gcpObj.DeepCopy()
		iamPolicy.Spec.ResourceReference = *attachmentPoint
		if attachmentPoint.Kind == "Project" {
			return &ast.NamespaceObject{
				FileObject: ast.FileObject{
					Object: iamPolicy,
					Source: o.Source,
				},
			}
		}

		ciam := &v1.ClusterIAMPolicy{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1.SchemeGroupVersion.String(),
				Kind:       v1.ClusterIAMPolicyKind,
			},
			ObjectMeta: iamPolicy.ObjectMeta,
			Spec:       iamPolicy.Spec,
			Status:     iamPolicy.Status,
		}
		v.addToClusterObjects(ciam, o.Source)
		return nil
	case *v1.OrganizationPolicy:
		if attachmentPoint == nil {
			panic(fmt.Sprintf("Missing attachment point for org policy %v", o))
		}
		orgPolicy := gcpObj.DeepCopy()
		orgPolicy.Spec.ResourceReference = *attachmentPoint
		// TODO(ttt): Is this enough attachment points? What about orgs/folders?
		if attachmentPoint.Kind == "Project" {
			return &ast.NamespaceObject{
				FileObject: ast.FileObject{
					Object: orgPolicy,
					Source: o.Source,
				},
			}
		}

		corg := &v1.ClusterOrganizationPolicy{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1.SchemeGroupVersion.String(),
				Kind:       v1.ClusterOrganizationPolicyKind,
			},
			ObjectMeta: orgPolicy.ObjectMeta,
			Spec:       orgPolicy.Spec,
			Status:     orgPolicy.Status,
		}
		v.addToClusterObjects(corg, o.Source)
		return nil
	}

	return o
}

// VisitObjectList implements Visitor.
func (v *GCPPolicyVisitor) VisitObjectList(o ast.ObjectList) ast.ObjectList {
	return v.Copying.VisitObjectList(o)
}

func (v *GCPPolicyVisitor) addToClusterObjects(o runtime.Object, source string) {
	co := &ast.ClusterObject{
		FileObject: ast.FileObject{
			Object: o,
			Source: source,
		},
	}
	v.cluster.Objects = append(v.cluster.Objects, co)
}
