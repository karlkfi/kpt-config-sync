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

package backend

import (
	"path/filepath"
	"time"

	"github.com/golang/glog"
	policyhierarchyv1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	contextCluster = "cluster"
	contextNode    = "node"
)

// OutputVisitor converts the AST into PolicyNode and ClusterPolicy objects.
type OutputVisitor struct {
	base *visitor.Base

	commitHash  string
	loadTime    time.Time
	allPolicies *policyhierarchyv1.AllPolicies
	context     string
	policyNode  []*policyhierarchyv1.PolicyNode
}

var _ ast.Visitor = &OutputVisitor{}

// NewOutputVisitor creates a new output visitor.
func NewOutputVisitor() *OutputVisitor {
	ov := &OutputVisitor{
		base: visitor.NewBase(),
	}
	ov.base.SetImpl(ov)
	return ov
}

// AllPolicies returns the AllPolicies object created by the visitor.
func (ov *OutputVisitor) AllPolicies() *policyhierarchyv1.AllPolicies {
	return ov.allPolicies
}

// VisitContext implements Visitor
func (ov *OutputVisitor) VisitContext(g *ast.Context) ast.Node {
	ov.allPolicies = &policyhierarchyv1.AllPolicies{
		PolicyNodes:   map[string]policyhierarchyv1.PolicyNode{},
		ClusterPolicy: &policyhierarchyv1.ClusterPolicy{},
	}
	ov.commitHash = g.ImportToken
	ov.loadTime = g.LoadTime
	ov.base.VisitContext(g)
	return nil
}

// VisitReservedNamespaces implements Visitor
func (ov *OutputVisitor) VisitReservedNamespaces(r *ast.ReservedNamespaces) ast.Node {
	for namespace := range r.ConfigMap.Data {
		ov.allPolicies.PolicyNodes[namespace] = policyhierarchyv1.PolicyNode{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
			Spec: policyhierarchyv1.PolicyNodeSpec{
				Type: policyhierarchyv1.ReservedNamespace,
			},
		}
	}
	return nil
}

// VisitCluster implements Visitor
func (ov *OutputVisitor) VisitCluster(c *ast.Cluster) ast.Node {
	ov.context = contextCluster
	ov.base.VisitCluster(c)
	return nil
}

// VisitTreeNode implements Visitor
func (ov *OutputVisitor) VisitTreeNode(n *ast.TreeNode) ast.Node {
	ov.context = contextNode
	origLen := len(ov.policyNode)
	var parent string
	if 0 < origLen {
		parent = ov.policyNode[origLen-1].Name
	}
	pn := &policyhierarchyv1.PolicyNode{
		ObjectMeta: metav1.ObjectMeta{
			Name:        filepath.Base(n.Path),
			Annotations: n.Annotations,
			Labels:      n.Labels,
		},
		Spec: policyhierarchyv1.PolicyNodeSpec{
			Parent: parent,
		},
	}
	if n.Type == ast.Namespace {
		pn.Spec.Type = policyhierarchyv1.Namespace
	} else {
		pn.Spec.Type = policyhierarchyv1.Policyspace
	}
	ov.policyNode = append(ov.policyNode, pn)
	ov.base.VisitTreeNode(n)
	ov.policyNode = ov.policyNode[:origLen]
	ov.allPolicies.PolicyNodes[pn.Name] = *pn
	return nil
}

// VisitObject implements Visitor
func (ov *OutputVisitor) VisitObject(o *ast.Object) ast.Node {
	switch ov.context {
	case contextCluster:
		spec := &ov.allPolicies.ClusterPolicy.Spec
		switch v := o.Object.(type) {
		case *rbacv1.ClusterRole:
			spec.ClusterRolesV1 = append(spec.ClusterRolesV1, *v)
		case *rbacv1.ClusterRoleBinding:
			spec.ClusterRoleBindingsV1 = append(spec.ClusterRoleBindingsV1, *v)
		case *extensionsv1beta1.PodSecurityPolicy:
			spec.PodSecurityPoliciesV1Beta1 = append(spec.PodSecurityPoliciesV1Beta1, *v)
		default:
			glog.Fatal("programmer error: invalid type %v in context %q", v, ov.context)
		}
	case contextNode:
		spec := &ov.policyNode[len(ov.policyNode)-1].Spec
		switch v := o.Object.(type) {
		case *rbacv1.Role:
			spec.RolesV1 = append(spec.RolesV1, *v)
		case *rbacv1.RoleBinding:
			spec.RoleBindingsV1 = append(spec.RoleBindingsV1, *v)
		case *corev1.ResourceQuota:
			spec.ResourceQuotaV1 = v
		default:
			glog.Fatal("programmer error: invalid type %v in context %q", v, ov.context)
		}
	default:
		glog.Fatal("programmer error: invalid context %q", ov.context)
	}
	return nil
}
