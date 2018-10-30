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
	"path"
	"time"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	v1alpha1 "github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/policyimporter/reserved"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	contextCluster = "cluster"
	contextNode    = "node"
)

// OutputVisitor converts the AST into PolicyNode and ClusterPolicy objects.
type OutputVisitor struct {
	*visitor.Base
	commitHash       string
	loadTime         time.Time
	allPolicies      *v1.AllPolicies
	context          string
	policyNode       []*v1.PolicyNode
	syncs            []*v1alpha1.Sync
	genericResources bool
}

var _ ast.Visitor = &OutputVisitor{}

// NewOutputVisitor creates a new output visitor.
func NewOutputVisitor(syncs []*v1alpha1.Sync, genericResources bool) *OutputVisitor {
	v := &OutputVisitor{Base: visitor.NewBase(), syncs: syncs, genericResources: genericResources}
	v.SetImpl(v)
	return v
}

// AllPolicies returns the AllPolicies object created by the visitor.
func (v *OutputVisitor) AllPolicies() *v1.AllPolicies {
	if v.genericResources {
		for _, s := range v.syncs {
			s.SetFinalizers(append(s.GetFinalizers(), v1alpha1.SyncFinalizer))
		}
	}
	v.allPolicies.Syncs = mapByName(v.syncs)
	return v.allPolicies
}

func mapByName(syncs []*v1alpha1.Sync) map[string]v1alpha1.Sync {
	m := make(map[string]v1alpha1.Sync)
	for _, sync := range syncs {
		m[sync.Name] = *sync
	}
	return m
}

// VisitRoot implements Visitor
func (v *OutputVisitor) VisitRoot(g *ast.Root) ast.Node {
	v.allPolicies = &v1.AllPolicies{
		PolicyNodes: map[string]v1.PolicyNode{},
		ClusterPolicy: &v1.ClusterPolicy{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1.SchemeGroupVersion.String(),
				Kind:       "ClusterPolicy",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: v1.ClusterPolicyName,
			},
		},
	}
	v.commitHash = g.ImportToken
	v.loadTime = g.LoadTime
	v.Base.VisitRoot(g)
	return nil
}

// VisitReservedNamespaces implements Visitor
func (v *OutputVisitor) VisitReservedNamespaces(r *ast.ReservedNamespaces) ast.Node {
	rns, err := reserved.From(&r.ConfigMap)
	if err != nil {
		panic(fmt.Sprintf("programmer error: input should have been validated %v", err))
	}
	for _, namespace := range rns.List(v1alpha1.ReservedAttribute) {
		v.allPolicies.PolicyNodes[namespace] = v1.PolicyNode{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1.SchemeGroupVersion.String(),
				Kind:       "PolicyNode",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
			Spec: v1.PolicyNodeSpec{
				Type: v1.ReservedNamespace,
			},
		}
	}
	return nil
}

// VisitCluster implements Visitor
func (v *OutputVisitor) VisitCluster(c *ast.Cluster) ast.Node {
	v.context = contextCluster
	v.Base.VisitCluster(c)
	return nil
}

// VisitTreeNode implements Visitor
func (v *OutputVisitor) VisitTreeNode(n *ast.TreeNode) ast.Node {
	v.context = contextNode
	origLen := len(v.policyNode)
	var name, parent string

	switch origLen {
	case 0:
		// root
		name = v1.RootPolicyNodeName
		parent = v1.NoParentNamespace
	case 1:
		name = path.Base(n.Path)
		parent = v1.RootPolicyNodeName
	default:
		name = path.Base(n.Path)
		parent = v.policyNode[origLen-1].Name
	}

	pn := &v1.PolicyNode{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       "PolicyNode",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Annotations: n.Annotations,
			Labels:      n.Labels,
		},
		Spec: v1.PolicyNodeSpec{
			Parent: parent,
		},
	}
	if n.Type == ast.Namespace {
		pn.Spec.Type = v1.Namespace
	} else {
		pn.Spec.Type = v1.Policyspace
	}
	v.policyNode = append(v.policyNode, pn)
	v.Base.VisitTreeNode(n)
	v.policyNode = v.policyNode[:origLen]
	v.allPolicies.PolicyNodes[pn.Name] = *pn
	return nil
}

// VisitClusterObject implements Visitor
func (v *OutputVisitor) VisitClusterObject(o *ast.ClusterObject) ast.Node {
	spec := &v.allPolicies.ClusterPolicy.Spec
	switch obj := o.FileObject.Object.(type) {
	case *rbacv1.ClusterRole:
		spec.ClusterRolesV1 = append(spec.ClusterRolesV1, *obj)
	case *rbacv1.ClusterRoleBinding:
		spec.ClusterRoleBindingsV1 = append(spec.ClusterRoleBindingsV1, *obj)
	case *extensionsv1beta1.PodSecurityPolicy:
		spec.PodSecurityPoliciesV1Beta1 = append(spec.PodSecurityPoliciesV1Beta1, *obj)
	}
	spec.Resources = appendResource(spec.Resources, o.FileObject.Object)
	return nil
}

// VisitObject implements Visitor
func (v *OutputVisitor) VisitObject(o *ast.NamespaceObject) ast.Node {
	spec := &v.policyNode[len(v.policyNode)-1].Spec
	switch obj := o.FileObject.Object.(type) {
	case *rbacv1.Role:
		spec.RolesV1 = append(spec.RolesV1, *obj)
	case *rbacv1.RoleBinding:
		spec.RoleBindingsV1 = append(spec.RoleBindingsV1, *obj)
	case *corev1.ResourceQuota:
		spec.ResourceQuotaV1 = obj
	}
	spec.Resources = appendResource(spec.Resources, o.FileObject.Object)
	return nil
}

// appendResource adds Object o to resources.
// GenericResources is grouped first by kind and then by version, and this method takes care of
// adding any required groupings for the new object, or adding to existing groupings if present.
func appendResource(resources []v1.GenericResources, o runtime.Object) []v1.GenericResources {
	gvk := o.GetObjectKind().GroupVersionKind()
	var gr *v1.GenericResources
	for i := range resources {
		if resources[i].Group == gvk.Group && resources[i].Kind == gvk.Kind {
			gr = &resources[i]
			break
		}
	}
	if gr == nil {
		resources = append(resources, v1.GenericResources{
			Group: gvk.Group,
			Kind:  gvk.Kind,
		})
		gr = &resources[len(resources)-1]
	}
	var gvr *v1.GenericVersionResources
	for i := range gr.Versions {
		if gr.Versions[i].Version == gvk.Version {
			gvr = &gr.Versions[i]
			break
		}
	}
	if gvr == nil {
		gr.Versions = append(gr.Versions, v1.GenericVersionResources{
			Version: gvk.Version,
		})
		gvr = &gr.Versions[len(gr.Versions)-1]
	}
	gvr.Objects = append(gvr.Objects, runtime.RawExtension{Object: o})
	return resources
}
