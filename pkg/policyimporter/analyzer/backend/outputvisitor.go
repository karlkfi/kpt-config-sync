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
	"fmt"
	"path/filepath"
	"time"

	"github.com/golang/glog"
	policyhierarchyv1 "github.com/google/nomos/pkg/api/nomos/v1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/policyimporter/reserved"
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
	v := &OutputVisitor{
		base: visitor.NewBase(),
	}
	v.base.SetImpl(v)
	return v
}

// AllPolicies returns the AllPolicies object created by the visitor.
func (v *OutputVisitor) AllPolicies() *policyhierarchyv1.AllPolicies {
	return v.allPolicies
}

// VisitContext implements Visitor
func (v *OutputVisitor) VisitContext(g *ast.Context) ast.Node {
	v.allPolicies = &policyhierarchyv1.AllPolicies{
		PolicyNodes: map[string]policyhierarchyv1.PolicyNode{},
		ClusterPolicy: &policyhierarchyv1.ClusterPolicy{
			TypeMeta: metav1.TypeMeta{
				APIVersion: policyhierarchyv1.SchemeGroupVersion.String(),
				Kind:       "ClusterPolicy",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: policyhierarchyv1.ClusterPolicyName,
			},
		},
	}
	v.commitHash = g.ImportToken
	v.loadTime = g.LoadTime
	v.base.VisitContext(g)
	return nil
}

// VisitReservedNamespaces implements Visitor
func (v *OutputVisitor) VisitReservedNamespaces(r *ast.ReservedNamespaces) ast.Node {
	rns, err := reserved.From(&r.ConfigMap)
	if err != nil {
		panic(fmt.Sprintf("programmer error: input should have been validated %v", err))
	}
	for _, namespace := range rns.List(policyhierarchyv1.ReservedAttribute) {
		v.allPolicies.PolicyNodes[namespace] = policyhierarchyv1.PolicyNode{
			TypeMeta: metav1.TypeMeta{
				APIVersion: policyhierarchyv1.SchemeGroupVersion.String(),
				Kind:       "PolicyNode",
			},
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
func (v *OutputVisitor) VisitCluster(c *ast.Cluster) ast.Node {
	v.context = contextCluster
	v.base.VisitCluster(c)
	return nil
}

// VisitTreeNode implements Visitor
func (v *OutputVisitor) VisitTreeNode(n *ast.TreeNode) ast.Node {
	v.context = contextNode
	origLen := len(v.policyNode)
	var parent string
	if 0 < origLen {
		parent = v.policyNode[origLen-1].Name
	}
	pn := &policyhierarchyv1.PolicyNode{
		TypeMeta: metav1.TypeMeta{
			APIVersion: policyhierarchyv1.SchemeGroupVersion.String(),
			Kind:       "PolicyNode",
		},
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
	v.policyNode = append(v.policyNode, pn)
	v.base.VisitTreeNode(n)
	v.policyNode = v.policyNode[:origLen]
	v.allPolicies.PolicyNodes[pn.Name] = *pn
	return nil
}

// VisitObjectList implements Visitor
func (v *OutputVisitor) VisitObjectList(o ast.ObjectList) ast.Node {
	return v.base.VisitObjectList(o)
}

// VisitObject implements Visitor
func (v *OutputVisitor) VisitObject(o *ast.Object) ast.Node {
	switch v.context {
	case contextCluster:
		spec := &v.allPolicies.ClusterPolicy.Spec
		switch obj := o.Object.(type) {
		case *rbacv1.ClusterRole:
			spec.ClusterRolesV1 = append(spec.ClusterRolesV1, *obj)
		case *rbacv1.ClusterRoleBinding:
			spec.ClusterRoleBindingsV1 = append(spec.ClusterRoleBindingsV1, *obj)
		case *extensionsv1beta1.PodSecurityPolicy:
			spec.PodSecurityPoliciesV1Beta1 = append(spec.PodSecurityPoliciesV1Beta1, *obj)
		default:
			glog.Fatalf("programmer error: invalid type %v in context %q", obj, v.context)
		}
	case contextNode:
		spec := &v.policyNode[len(v.policyNode)-1].Spec
		switch obj := o.Object.(type) {
		case *rbacv1.Role:
			spec.RolesV1 = append(spec.RolesV1, *obj)
		case *rbacv1.RoleBinding:
			spec.RoleBindingsV1 = append(spec.RoleBindingsV1, *obj)
		case *corev1.ResourceQuota:
			spec.ResourceQuotaV1 = obj
		default:
			glog.Fatalf("programmer error: invalid type %v in context %q", obj, v.context)
		}
	default:
		glog.Fatalf("programmer error: invalid context %q", v.context)
	}
	return nil
}
