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

// Package transform provides the functions to traverse through an *ast.Root
// tree. A valid Bespin-managed GCP top-level directory (i.e. namespaces/*)
// should contain only root directories of each GCP root resources, such that
// multi-organization, multi-folder and single/multi-project management are all
// supported. For example, below is a valid GCP hierarchy that Bespin should be
// able to manage:
//
// ├── system
// |   └── bespin.yaml
// ├── namespaces  <-- this is the entry of our GCP hierarchy
// │   ├── folder1  <-- this represents a root-directory, which manages everything for GCP folder1
// │   │   └── folder.yaml
// │   ├── folder2  <-- another root-directory.
// │   │   ├── folder3
// │   │   │   └── folder.yaml
// │   │   ├── folder.yaml
// │   │   └── project2
// │   │       └── project.yaml
// │   ├── organization1
// │   │   ├── folder4
// │   │   │   └── folder.yaml
// │   │   ├── folder5
// │   │   │   ├── folder6
// │   │   │   │   └── folder.yaml
// │   │   │   ├── folder.yaml
// │   │   │   └── project3
// │   │   │       └── project.yaml
// │   │   ├── organization.yaml
// │   │   └── org_policy.yaml
// │   └── project1
// │       ├── iam.yaml
// │       └── project.yaml
package transform

import (
	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/policyascode/v1"
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

// Error implements Visitor.
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

// gcpHierarchyContext stores the objects seen as the tree is traversed.
type gcpHierarchyContext struct {
	prev *gcpHierarchyContext
	// There should be at most 1 cluster level object per TreeNode.
	// It can be either Organization or Folder.
	clusterObj *ast.ClusterObject
	// Policy attachment point associated with the current TreeNode.
	policyAttachmentPoint *v1.ResourceReference
}

// needsAttachmentPoint returns true if the visitor's current context is visiting
// a TreeNode that MUST have an attachment point. GCP organiztion/folder/project
// are valid attachment points, and bespin structure requires:
// 1. "namespaces/" TreeNode, the top TreeNode of the repo, should have NO attachment
//     point;
// 2. Each directory/sub-directory TreeNode under "namespaces/" should have exactly
//    one attachment point.
func (c *gcpHierarchyContext) needsAttachmentPoint() bool {
	// Only "namespaces/" TreeNode has no prev context.
	return c.prev != nil
}

// needsNamespace returns true if the visitor's current context is inside a GCP
// hierarchy and has no cluster scope resources.
func (c *gcpHierarchyContext) needsNamespace() bool {
	return c.prev != nil && c.clusterObj == nil
}

type gcpAttachmentPointKeyType struct{}

// Extension key storing/retrieving the policy attachment point resource
// reference. There is at most 1 policy attachment point in each TreeNode.
var gcpAttachmentPointKey = gcpAttachmentPointKeyType{}

// VisitTreeNode visits a tree node.
// If the object is a folder or organization, it is added as a cluster level
// resource instead.
func (v *GCPHierarchyVisitor) VisitTreeNode(n *ast.TreeNode) *ast.TreeNode {
	// Top-level directory (i.e. namespaces/*) should contain only root
	// directories of each GCP root resources.
	if v.ctx == nil {
		if len(n.Objects) > 0 {
			var objNames []string
			for _, obj := range n.Objects {
				objNames = append(objNames, obj.RelativeSlashPath())
			}
			v.errs.Add(errors.Errorf("GCP top-level hierarchy should not contain specific resources. Found %v", objNames))
			return nil
		}
	}

	ctx := &gcpHierarchyContext{
		prev: v.ctx,
	}
	v.ctx = ctx

	// Call c.Copying.VisitTreeNode to continue iteration.
	newNode := v.Copying.VisitTreeNode(n)

	if v.ctx.needsAttachmentPoint() && v.ctx.policyAttachmentPoint == nil {
		v.errs.Add(errors.Errorf("Missing GCP policy attachment point, must be an organization, folder, or project"))
		return nil
	}
	if v.ctx.needsNamespace() {
		glog.V(1).Infof("Marking tree node %v as namespace scope", newNode.Path)
		newNode.Type = ast.Namespace
	}
	if v.ctx.clusterObj != nil {
		glog.V(1).Infof("Moving %v to cluster scope", v.ctx.clusterObj.RelativeSlashPath())
		v.cluster.Objects = append(v.cluster.Objects, v.ctx.clusterObj)
	}
	if v.ctx.policyAttachmentPoint != nil {
		newNode.Data = newNode.Data.Add(gcpAttachmentPointKey, v.ctx.policyAttachmentPoint)
	}

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
		p := gcpObj.DeepCopy()
		pr := v.parentReference()
		// Project without parent must be at root.
		if pr == nil && !v.visitingRoot() {
			v.errs.Add(errors.Errorf("Project %v without a folder or organization parent must be at root", gcpObj))
			return nil
		}
		if pr != nil {
			if pr.Kind != "Folder" && pr.Kind != "Organization" {
				v.errs.Add(errors.Errorf("Project %v must have either folder or organization as its parent: %v", gcpObj, pr))
				return nil
			}
			p.Spec.ParentReference = *pr
		}
		return &ast.NamespaceObject{
			FileObject: ast.NewFileObject(p, o.RelativeSlashPath()),
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
			if !v.visitingRoot() {
				v.errs.Add(errors.Errorf("Folder %v without a folder or organization parent must be at root", gcpObj))
				return nil
			}
			v.ctx.clusterObj = &ast.ClusterObject{
				FileObject: o.FileObject,
			}
			return nil
		}
		if pr.Kind != "Folder" && pr.Kind != "Organization" {
			v.errs.Add(errors.Errorf("Folder %v must have either folder or organization as its parent: %v", gcpObj, pr))
			return nil
		}
		f := gcpObj.DeepCopy()
		f.Spec.ParentReference = *pr
		v.ctx.clusterObj = &ast.ClusterObject{
			FileObject: ast.NewFileObject(f, o.RelativeSlashPath()),
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
		if !v.visitingRoot() {
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

// visitingRoot returns true if the visitor is currently visiting a root tree node.
func (v *GCPHierarchyVisitor) visitingRoot() bool {
	return v.ctx != nil && v.ctx.prev != nil && v.ctx.prev.prev == nil
}
