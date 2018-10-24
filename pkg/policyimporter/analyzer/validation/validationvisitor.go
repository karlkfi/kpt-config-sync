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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// InputValidator checks various filesystem constraints after loading into the tree format.
// Error messages emitted from the validator should be formatted to first print the constraint
// that is being violated, then print a useful error message on what is voilating the constraint
// and what is required to fix it.
type InputValidator struct {
	base               *visitor.Base
	errs               *multierror.Builder
	reserved           *reserved.Namespaces
	names              map[string]*ast.TreeNode
	nodes              []*ast.TreeNode
	seenResourceQuotas map[string]struct{}
	allowedGVKs        map[schema.GroupVersionKind]struct{}
}

// InputValidator implements ast.Visitor
var _ ast.Visitor = &InputValidator{}

// NewInputValidator creates a new validator
func NewInputValidator(allowedGVKs map[schema.GroupVersionKind]struct{}) *InputValidator {
	v := &InputValidator{
		base:               visitor.NewBase(),
		errs:               multierror.NewBuilder(),
		reserved:           reserved.EmptyNamespaces(),
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
func (v *InputValidator) VisitReservedNamespaces(r *ast.ReservedNamespaces) ast.Node {
	reserved, err := reserved.From(&r.ConfigMap)
	if err != nil {
		v.errs.Add(err)
	} else {
		v.reserved = reserved
	}
	return nil
}

// VisitCluster implements Visitor
func (v *InputValidator) VisitCluster(c *ast.Cluster) ast.Node {
	return v.base.VisitCluster(c)
}

// VisitTreeNode implements Visitor
func (v *InputValidator) VisitTreeNode(n *ast.TreeNode) ast.Node {
	name := path.Base(n.Path)
	if v.reserved.IsReserved(name) {
		v.errs.Add(errors.Errorf(
			"Reserved namespaces must not be used for %s names.  "+
					"Directory %q declares a %s which conflicts with a reserved namespace name. "+
					"Adjust the directory name for %q or remove %s from the reserved namespace config.",
			n.Type, n.Path, n.Type, n.Path, path.Base(n.Path)))
	}
	if other, found := v.names[name]; found {
		v.errs.Add(errors.Errorf(
			"Names for %s must not match names for other %ss.  "+
					"Declaration in directory %q duplicates name from declaration in %q. "+
					"Adjust one of the directory names.",
			n.Type, other.Type, n.Path, other.Path))
	}
	if len(v.nodes) != 0 {
		if parent := v.nodes[len(v.nodes)-1]; parent.Type == ast.Namespace {
			v.errs.Add(errors.Errorf(
				"Namespaces must not contain children.  "+
						"Namespace declared in directory %q cannot have child declared in subdirectory %q. "+
						"Restructure directories so namespace %q does not have children.",
				parent.Path, n.Path, path.Base(n.Path)))
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

// VisitClusterObject implements Visitor
func (v *InputValidator) VisitClusterObject(o *ast.ClusterObject) ast.Node {
	gvk := o.FileObject.GetObjectKind().GroupVersionKind()

	metaObj := o.ToMeta()
	ns := metaObj.GetNamespace()
	if ns != "" {
		v.errs.Add(errors.Errorf(
			"Cluster scoped objects must not be associated with a namespace. "+
					"Remove the namespace field from object.  "+
					"Object %s, Name=%q is declared with namespace %s",
			gvk,
			metaObj.GetName(),
			ns))
	}

	if gvk == corev1.SchemeGroupVersion.WithKind("Namespace") {
		v.errs.Add(errors.Errorf(
			"Cannot declare namespaces in %q directory.  Namespaces must be declared in a "+
					"%q directory in the hierarchy. "+
					"Remove namespace %s in file %s",
			repo.ClusterDir,
			repo.NamespacesDir,
			metaObj.GetName(),
			o.Source))
	}

	if _, found := v.allowedGVKs[gvk]; !found {
		v.errs.Add(errors.Errorf("Sync for objects of type %#v is not enabled. Remove object "+
				"%s in file %s, or add a Sync for that type.", gvk, metaObj.GetName(),
			o.Source))
	}

	v.checkAnnotationsAndLabels(metaObj, o.Source)

	return nil
}

// VisitObjectList implements Visitor
func (v *InputValidator) VisitObjectList(o ast.ObjectList) ast.Node {
	return v.base.VisitObjectList(o)
}

// VisitObject implements Visitor
func (v *InputValidator) VisitObject(o *ast.NamespaceObject) ast.Node {
	v.checkSingleResourceQuota(o)
	metaObj := o.ToMeta()
	ns := metaObj.GetNamespace()
	node := v.nodes[len(v.nodes)-1]

	gvk := o.GetObjectKind().GroupVersionKind()
	if _, found := v.allowedGVKs[gvk]; !found {
		v.errs.Add(errors.Errorf("Sync for objects of type %#v is not enabled. Remove object "+
				"%s in file %s, or add a Sync for that type.", gvk, metaObj.GetName(),
			o.Source))
	}

	if ns != "" {
		if node.Type == ast.Policyspace {
			v.errs.Add(errors.Errorf(
				"Objects declared in abstract namespaces directories must not have a namespace specified. "+
						"Remove the namespace field from object.  "+
						"Directory %q has declaration for %s, name %q with namespace %s",
				node.Path,
				o.FileObject.GetObjectKind().GroupVersionKind(),
				metaObj.GetName(),
				ns))
		}
	}
	if nodeNS := path.Base(node.Path); nodeNS != ns && node.Type == ast.Namespace {
		v.errs.Add(errors.Errorf("Object's Namespace must match the name of the namespace "+
				"directory in which the object appears. Object Namespace is %s. Directory name is %s. "+
				"object: %#v",
			ns, nodeNS, o.FileObject))
	}

	if node.Type == ast.Policyspace {
		switch gvk {
		case rbacv1.SchemeGroupVersion.WithKind("RoleBinding"):
		case corev1.SchemeGroupVersion.WithKind("ResourceQuota"):
		default:
			v.errs.Add(errors.Errorf(
				"Objects of type %s are not allowed in abstract namespace directories.  Move %q to a namespace "+
						"directory",
				gvk,
				metaObj.GetName(),
			))
		}
	}

	v.checkAnnotationsAndLabels(metaObj, o.Source)

	return nil
}

// checkSingleResourceQuota ensures that at most one ResourceQuota object is present in each
// directory.
func (v *InputValidator) checkSingleResourceQuota(o *ast.NamespaceObject) {
	if o.FileObject.GetObjectKind().GroupVersionKind() != corev1.SchemeGroupVersion.WithKind("ResourceQuota") {
		return
	}
	path := v.nodes[len(v.nodes)-1].Path
	if _, found := v.seenResourceQuotas[path]; found {
		v.errs.Add(errors.Errorf("Each directory must contain at most one ResourceQuota object. "+
				"Object name: \"%s\", found at path \"%s\".", o.ToMeta().GetName(), path))
	} else {
		v.seenResourceQuotas[path] = struct{}{}
	}
}

func (v *InputValidator) checkAnnotationsAndLabels(o metav1.Object, source string) {
	if err := v.checkAnnotations(o, source); err != nil {
		v.errs.Add(err)
	}
	if err := v.checkLabels(o, source); err != nil {
		v.errs.Add(err)
	}
}

func (v *InputValidator) checkAnnotations(o metav1.Object, source string) error {
	return checkNomosPrefix(
		o.GetAnnotations(),
		v1alpha1.InputAnnotations,
		"Objects are not allowed to define unsupported annotations starting with \"nomos.dev/\". "+
				"Object %s defined in %q has offending annotations: %s",
		o,
		source,
	)
}

var ignoreNone = map[string]struct{}{}

func (v *InputValidator) checkLabels(o metav1.Object, source string) error {
	return checkNomosPrefix(
		o.GetLabels(),
		ignoreNone,
		"Objects are not allowed to define labels starting with \"nomos.dev/\". "+
				"Object %s defined in %q has %s",
		o,
		source,
	)
}

func checkNomosPrefix(m map[string]string, ignore map[string]struct{}, errFmt string, o metav1.Object, source string) error {
	var found []string
	for k, v := range m {
		if _, found := ignore[k]; found {
			continue
		}
		if strings.HasPrefix(k, policyhierarchy.GroupName+"/") {
			found = append(found, fmt.Sprintf("%s=%s", k, v))
		}
	}
	if len(found) == 0 {
		return nil
	}
	return errors.Errorf(
		errFmt,
		o.GetName(),
		source,
		strings.Join(found, ", "))
}
