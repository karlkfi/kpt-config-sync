/*
Copyright 2017 The Nomos Authors.
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

package validation

import (
	"fmt"
	"path"
	"strings"

	"github.com/google/nomos/pkg/api/policyhierarchy"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/policyimporter/reserved"
	"github.com/google/nomos/pkg/util/multierror"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// InputValidator checks various filesystem constraints after loading into the tree format.
// Error messages emitted from the validator should be formatted to first print the constraint
// that is being violated, then print a useful error message on what is violating the constraint
// and what is required to fix it.
type InputValidator struct {
	base               *visitor.Base
	errs               multierror.Builder
	reserved           *reserved.Namespaces
	dirNames           map[string]*ast.TreeNode
	nodes              []*ast.TreeNode
	seenResourceQuotas map[string]struct{}
	allowedGVKs        map[schema.GroupVersionKind]struct{}
}

// InputValidator implements ast.Visitor
var _ ast.Visitor = &InputValidator{}

// NewInputValidator creates a new validator.  allowedGVKs represents the set
// of valid group-version-kinds for objects in the namespaces and cluster
// directories.  Objects of other types will be treated as an error.
func NewInputValidator(allowedGVKs map[schema.GroupVersionKind]struct{}) *InputValidator {
	v := &InputValidator{
		base:               visitor.NewBase(),
		reserved:           reserved.EmptyNamespaces(),
		dirNames:           make(map[string]*ast.TreeNode),
		seenResourceQuotas: make(map[string]struct{}),
		allowedGVKs:        allowedGVKs,
	}
	v.base.SetImpl(v)
	return v
}

// Error returns any errors encountered during processing
func (v *InputValidator) Error() error {
	return v.errs.Build()
}

// VisitRoot implements Visitor
func (v *InputValidator) VisitRoot(g *ast.Root) ast.Node {
	v.base.VisitRoot(g)
	return g
}

// VisitReservedNamespaces implements Visitor
func (v *InputValidator) VisitReservedNamespaces(rs *ast.ReservedNamespaces) ast.Node {
	r, err := reserved.From(&rs.ConfigMap)
	if err != nil {
		v.errs.Add(err)
	} else {
		v.reserved = r
	}
	return nil
}

// VisitCluster implements Visitor
func (v *InputValidator) VisitCluster(c *ast.Cluster) ast.Node {
	return v.base.VisitCluster(c)
}

// An error factory for this treeNode.
type treeNodeError struct {
	ast.TreeNode
}

func (n treeNodeError) hasReservedName() error {
	return errors.Errorf("Directories may not have reserved namespace names. "+
		"Rename the below directory or remove %[1]q from %[4]s/%[3]s:\n"+
		"%[2]s",
		n.Name(), n, v1alpha1.ReservedNamespacesConfigMapName, repo.SystemDir)
}

func (n treeNodeError) duplicates(other *ast.TreeNode) error {
	return errors.Errorf("Directory names must be unique. "+
		"The two directories below share the same name. Rename one:\n"+
		"%[1]s\n\n"+
		"%[2]s",
		n, other)
}

func (n treeNodeError) hasParent(parent *ast.TreeNode) error {
	return errors.Errorf("A %[1]s directory may not have children. "+
		"Restructure %[4]s so that it does not have the child %[2]q:\n"+
		"%[3]s",
		ast.Namespace, n.Name(), n, parent.Name())
}

func (n treeNodeError) usesNamespaceSelectorAnnotation() error {
	return errors.Errorf("A %[3]s may not use the annotation %[2]s. "+
		"Remove metadata.annotations.%[2]s from:\n"+
		"%[1]s",
		n, v1alpha1.NamespaceSelectorAnnotationKey, ast.Namespace)
}

// VisitTreeNode implements Visitor
func (v *InputValidator) VisitTreeNode(n *ast.TreeNode) ast.Node {
	name := path.Base(n.Path)
	nodeError := treeNodeError{*n}
	if v.reserved.IsReserved(name) {
		// The namespace's name must not be a reserved namespace name.
		v.errs.Add(nodeError.hasReservedName())
	}
	if other, found := v.dirNames[name]; found {
		// The namespace must not duplicate the name of another namespace.
		v.errs.Add(nodeError.duplicates(other))
	} else {
		v.dirNames[name] = n
	}
	if len(v.nodes) != 0 {
		// Namespaces may not have children.
		if parent := v.nodes[len(v.nodes)-1]; parent.Type == ast.Namespace {
			v.errs.Add(nodeError.hasParent(parent))
		}
	}
	if n.Type == ast.Namespace {
		if _, found := n.Annotations[v1alpha1.NamespaceSelectorAnnotationKey]; found {
			// Namespaces may not use the selector annotation.
			v.errs.Add(nodeError.usesNamespaceSelectorAnnotation())
		}
	}

	v.nodes = append(v.nodes, n)
	v.base.VisitTreeNode(n)
	v.nodes = v.nodes[:len(v.nodes)-1]
	return nil
}

// VisitClusterObjectList implements Visitor
func (v *InputValidator) VisitClusterObjectList(o ast.ClusterObjectList) ast.Node {
	return v.base.VisitClusterObjectList(o)
}

func isNotSyncable(o ast.FileObject) error {
	return errors.Errorf(
		"The below resource is not syncable. Enable sync for resources of type %[1]q.\n"+
			"%[2]s",
		o.GroupVersionKind(), o)
}

func (v *InputValidator) checkAnnotationsAndLabels(o ast.FileObject) {
	if err := v.checkAnnotations(o); err != nil {
		v.errs.Add(err)
	}
	if err := v.checkLabels(o); err != nil {
		v.errs.Add(err)
	}
}

// VisitClusterObject implements Visitor
func (v *InputValidator) VisitClusterObject(o *ast.ClusterObject) ast.Node {
	gvk := o.GroupVersionKind()
	if _, found := v.allowedGVKs[gvk]; !found {
		v.errs.Add(isNotSyncable(o.FileObject))
	}

	v.checkAnnotationsAndLabels(o.FileObject)

	return nil
}

// VisitObjectList implements Visitor
func (v *InputValidator) VisitObjectList(o ast.ObjectList) ast.Node {
	return v.base.VisitObjectList(o)
}

// An error factory for this NamespaceObject
type namespaceObjectErrorFactory ast.NamespaceObject

func (o namespaceObjectErrorFactory) hasMultipleResourceQuotas() error {
	return errors.Errorf(
		"A directory may contain at most one ResourceQuota resource. "+
			"The %[1]q directory contains multiple ResourceQuota resource definitions, including the below:\n"+
			"%[2]s",
		path.Dir(o.Source), o)
}

func (o namespaceObjectErrorFactory) hasNamespace() error {
	return errors.Errorf(
		"%[1]s scoped resources may not declare metadata.namespace. Remove the metadata.namespace field from:\n"+
			"%[2]s",
		ast.AbstractNamespace, o)
}

func (o namespaceObjectErrorFactory) hasWrongNamespace(namespace string) error {
	return errors.Errorf(
		"%[1]s scoped resources must declare a metadata.namespace identical to the directory in which they are defined.\n"+
			"Expected Namespace: %[3]s\n"+
			"Actual Namespace:   %[4]s\n"+
			"\n%[2]s",
		ast.Namespace, o, namespace, path.Base(path.Dir(o.Source)))
}

func (o namespaceObjectErrorFactory) hasIllegalType() error {
	return errors.Errorf(
		"Resorces of type %[1]q are not allowed in %[2]s directories. "+
			"Move the below resource to a %[3]s directory:\n"+
			"%[4]s",
		o.GroupVersionKind(), ast.AbstractNamespace, ast.Namespace, o)
}

// VisitObject implements Visitor
func (v *InputValidator) VisitObject(o *ast.NamespaceObject) ast.Node {
	nodeError := namespaceObjectErrorFactory(*o)

	// checkSingleResourceQuota ensures that at most one ResourceQuota resource is present in each
	// directory.
	if o.GroupVersionKind() == corev1.SchemeGroupVersion.WithKind("ResourceQuota") {
		curPath := v.nodes[len(v.nodes)-1].Path
		if _, found := v.seenResourceQuotas[curPath]; found {
			v.errs.Add(nodeError.hasMultipleResourceQuotas())
		} else {
			v.seenResourceQuotas[curPath] = struct{}{}
		}
	}

	if _, found := v.allowedGVKs[o.GroupVersionKind()]; !found {
		v.errs.Add(isNotSyncable(o.FileObject))
	}

	namespace := o.ToMeta().GetNamespace()
	node := v.nodes[len(v.nodes)-1]
	if namespace != "" && node.Type == ast.AbstractNamespace {
		v.errs.Add(nodeError.hasNamespace())
	}
	if nodeNS := path.Base(node.Path); nodeNS != namespace && node.Type == ast.Namespace {
		v.errs.Add(nodeError.hasWrongNamespace(namespace))
	}

	if node.Type == ast.AbstractNamespace {
		switch o.GroupVersionKind() {
		case rbacv1.SchemeGroupVersion.WithKind("RoleBinding"):
		case corev1.SchemeGroupVersion.WithKind("ResourceQuota"):
		default:
			v.errs.Add(nodeError.hasIllegalType())
		}
	}

	v.checkAnnotationsAndLabels(o.FileObject)

	return nil
}

var ignoreNone = map[string]struct{}{}

func (v *InputValidator) checkAnnotations(o ast.FileObject) error {
	return checkNomosPrefix(
		o.ToMeta().GetAnnotations(),
		v1alpha1.InputAnnotations,
		"Objects are not allowed to define unsupported annotations starting with \"nomos.dev/\". "+
			"The below object has offending annotations: %s\n%s",
		o)
}

func (v *InputValidator) checkLabels(o ast.FileObject) error {
	return checkNomosPrefix(
		o.ToMeta().GetLabels(),
		ignoreNone,
		"Objects are not allowed to define labels starting with \"nomos.dev/\". "+
			"The below object has offending labels: %s\n%s",
		o)
}

func checkNomosPrefix(m map[string]string, ignore map[string]struct{}, errFmt string, o ast.FileObject) error {
	var found []string
	for k := range m {
		if _, found := ignore[k]; found {
			continue
		}
		if strings.HasPrefix(k, policyhierarchy.GroupName+"/") {
			found = append(found, fmt.Sprintf("%q", k))
		}
	}
	if len(found) == 0 {
		return nil
	}
	return errors.Errorf(
		errFmt,
		strings.Join(found, ", "),
		o)
}
