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
	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/policyascode/v1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/util/multierror"
	"github.com/pkg/errors"
)

// GCPHierarchyVisitor sets up the hierarchical relationship between
// GCP resources. It does the following:
// * Move organization and folder from namespace scope to cluster scope.
// * Set up parent reference between project, folder, and organization.
// * Store resource reference to the project/folder/organization as extension
//   data in each TreeNode to be used by policy visitors.
type GCPHierarchyVisitor struct {
	*visitor.Copying
	cluster *ast.Cluster
	ctx     *gcpHierarchyContext
	// cumulative errors encountered by the visitor
	errs multierror.Builder
}

var _ ast.Visitor = &GCPHierarchyVisitor{}

// NewGCPHierarchyVisitor makes a new visitor.
func NewGCPHierarchyVisitor() *GCPHierarchyVisitor {
	v := &GCPHierarchyVisitor{Copying: visitor.NewCopying()}
	v.SetImpl(v)
	return v
}

// Error implements CheckingVisitor.
// The error checking in this visitor is rudimentary.
// The expectation is that a thorough error checking is done by another
// validation visitor.
func (v *GCPHierarchyVisitor) Error() error {
	return v.errs.Build()
}

// VisitCluster implements Visitor.
func (v *GCPHierarchyVisitor) VisitCluster(c *ast.Cluster) *ast.Cluster {
	newC := v.Copying.VisitCluster(c)
	v.cluster = newC
	return newC
}

// VisitReservedNamespaces implements Visitor. Currently unused and always returns
// the passed node.
func (v *GCPHierarchyVisitor) VisitReservedNamespaces(r *ast.ReservedNamespaces) *ast.ReservedNamespaces {
	return r
}

// gcpHierarchyContext stores the objects seen as the tree is traversed.
type gcpHierarchyContext struct {
	prev *gcpHierarchyContext
	// There should be at most 1 cluster level object per TreeNode.
	// It can be either Organization or Folder.
	clusterObj *ast.ClusterObject
	// Policy attachment point associated with the current TreeNode.
	policyAttachmentPoint *v1.ResourceReference
}

type gcpAttachmentPointKeyType struct{}

// Extension key storing/retrieving the policy attachment point resource
// reference. There is at most 1 policy attachment point in each TreeNode.
var gcpAttachmentPointKey = gcpAttachmentPointKeyType{}

// VisitTreeNode visits a tree node.
// If the object is a folder or organization, it is added as a cluster level
// resource instead.
func (v *GCPHierarchyVisitor) VisitTreeNode(n *ast.TreeNode) *ast.TreeNode {
	ctx := &gcpHierarchyContext{
		prev: v.ctx,
	}
	v.ctx = ctx

	// Call c.Copying.VisitTreeNode to continue iteration.
	newNode := v.Copying.VisitTreeNode(n)

	if v.ctx.clusterObj != nil {
		glog.V(1).Infof("Moving %v to cluster scope", v.ctx.clusterObj.Source)
		v.cluster.Objects = append(v.cluster.Objects, v.ctx.clusterObj)
	}
	newNode.Data = newNode.Data.Add(gcpAttachmentPointKey, v.ctx.policyAttachmentPoint)

	v.ctx = ctx.prev
	return newNode
}

// VisitObject sets up parent references and removes folder/organization from
// namespace object list.
func (v *GCPHierarchyVisitor) VisitObject(o *ast.NamespaceObject) *ast.NamespaceObject {
	gvk := o.GetObjectKind().GroupVersionKind()
	if gvk.Group != v1.SchemeGroupVersion.Group {
		return o
	}
	switch gcpObj := o.FileObject.Object.(type) {
	case *v1.Project:
		if !v.setAttachmentPoint(o) {
			return nil
		}
		pr := v.parentReference()
		if pr == nil {
			v.errs.Add(errors.Errorf("GCP project %v is missing its parent.", gcpObj.Name))
			return nil
		}
		if pr.Kind != "Folder" && pr.Kind != "Organization" {
			v.errs.Add(errors.Errorf("Project %v must have either folder or org as its parent: %v", gcpObj, pr))
			return nil
		}
		p := gcpObj.DeepCopy()
		p.Spec.ParentReference = *pr
		return &ast.NamespaceObject{
			FileObject: ast.FileObject{
				Object: p,
				Source: o.Source,
			},
		}
	case *v1.Folder:
		if !v.setAttachmentPoint(o) {
			return nil
		}
		if v.ctx.clusterObj != nil {
			v.errs.Add(errors.Errorf("Invalid hierarchy: %v and %v ", v.ctx.clusterObj, gcpObj))
			return nil
		}
		pr := v.parentReference()
		if pr == nil {
			if v.ctx.prev == nil {
				v.ctx.clusterObj = &ast.ClusterObject{
					FileObject: o.FileObject,
				}
			} else {
				v.errs.Add(errors.Errorf("Folder %v is missing its parent", gcpObj))
			}
			return nil
		}
		if pr.Kind != "Folder" && pr.Kind != "Organization" {
			v.errs.Add(errors.Errorf("Folder %v must have either folder or org as its parent: %v", gcpObj, pr))
			return nil
		}
		f := gcpObj.DeepCopy()
		f.Spec.ParentReference = *pr
		v.ctx.clusterObj = &ast.ClusterObject{
			FileObject: ast.FileObject{
				Object: f,
				Source: o.Source,
			},
		}
		return nil
	case *v1.Organization:
		if !v.setAttachmentPoint(o) {
			return nil
		}
		if v.ctx.clusterObj != nil {
			v.errs.Add(errors.Errorf("Invalid hierarchy: %v and %v ", v.ctx.clusterObj, gcpObj))
			return nil
		}
		if v.ctx.prev != nil {
			v.errs.Add(errors.Errorf("Organization must be at root level"))
			return nil
		}
		v.ctx.clusterObj = &ast.ClusterObject{
			FileObject: o.FileObject,
		}
		return nil
	}

	return o
}

// setAttachmentPoint set the attachment point in the current context to the
// given namespace object. It returns true on success.
func (v *GCPHierarchyVisitor) setAttachmentPoint(o *ast.NamespaceObject) bool {
	if v.ctx.policyAttachmentPoint != nil {
		v.errs.Add(errors.Errorf("Too many GCP policy attachment points: %v", v.ctx.policyAttachmentPoint))
		return false
	}
	v.ctx.policyAttachmentPoint = &v1.ResourceReference{
		Kind: o.GroupVersionKind().Kind,
		Name: o.Name(),
	}
	return true
}

// parentReference returns the name and kind that point to the parent GCP
// folder/organization, or nil if there is no parent.
func (v *GCPHierarchyVisitor) parentReference() *v1.ParentReference {
	if v.ctx.prev == nil || v.ctx.prev.clusterObj == nil {
		return nil
	}
	gvk := v.ctx.prev.clusterObj.GetObjectKind().GroupVersionKind()
	return &v1.ParentReference{
		Kind: gvk.Kind,
		Name: v.ctx.prev.clusterObj.Name(),
	}
}

// VisitObjectList visits the object list.
func (v *GCPHierarchyVisitor) VisitObjectList(o ast.ObjectList) ast.ObjectList {
	return v.Copying.VisitObjectList(o)
}
