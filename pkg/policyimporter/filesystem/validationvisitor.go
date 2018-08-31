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

package filesystem

import (
	"path/filepath"

	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/policyimporter/reserved"
	"github.com/google/nomos/pkg/syncer/multierror"
	"github.com/pkg/errors"
)

// InputValidator checks various filesystem constraints after loading into the tree format.
// Error messages emitted from the validator should be formatted to first print the constraint
// that is being violated, then print a useful error message on what is voilating the constraint
// and what is required to fix it.
type InputValidator struct {
	base     *visitor.Base
	errs     *multierror.Builder
	reserved *reserved.Namespaces
	names    map[string]*ast.TreeNode
	nodes    []*ast.TreeNode
}

// InputValidator implements ast.Visitor
var _ ast.Visitor = &InputValidator{}

// NewInputValidator creates a new validator
func NewInputValidator() *InputValidator {
	v := &InputValidator{
		base:     visitor.NewBase(),
		errs:     multierror.NewBuilder(),
		reserved: reserved.EmptyNamespaces(),
	}
	v.base.SetImpl(v)
	return v
}

// Result returns any errors encountered during processing
func (v *InputValidator) Result() error {
	return v.errs.Build()
}

// VisitContext implements Visitor
func (v *InputValidator) VisitContext(g *ast.Context) ast.Node {
	v.base.VisitContext(g)
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
	name := filepath.Base(n.Path)
	if v.reserved.IsReserved(name) {
		v.errs.Add(errors.Errorf(
			"Reserved namespaces cannot be used for %s names.  "+
				"Directory %q declares a %s which conflicts with a reserved namespace name, "+
				"adjust the directory name for %q or remove %s from the reserved namespace config to resolve.",
			n.Type, n.Path, n.Type, n.Path, filepath.Base(n.Path)))
	}
	if other, found := v.names[name]; found {
		v.errs.Add(errors.Errorf(
			"Names for %s cannot duplicate names for %ss.  "+
				"Declaration in directory %q duplicates name from declaration in %q, "+
				"adjust one of the directory names to resolve.",
			n.Type, other.Type, n.Path, other.Path))
	}

	if len(v.nodes) != 0 {
		if parent := v.nodes[len(v.nodes)-1]; parent.Type == ast.Namespace {
			v.errs.Add(errors.Errorf(
				"Namespaces cannot contain children.  "+
					"Namespace declared in directory %q cannot have child declared in subdirectory %q, "+
					"restructure directories so namespace %q does not have children to resolve.",
				parent.Path, n.Path, filepath.Base(n.Path)))
		}
	}

	v.nodes = append(v.nodes, n)
	v.base.VisitTreeNode(n)
	v.nodes = v.nodes[:len(v.nodes)-1]
	return nil
}

// VisitObject implements Visitor
func (v *InputValidator) VisitObject(o *ast.Object) ast.Node {
	metaObj := o.ToMeta()
	if ns := metaObj.GetNamespace(); ns != "" {
		if len(v.nodes) == 0 {
			v.errs.Add(errors.Errorf(
				"Cluster scoped objects are not associated with a namespace, "+
					"remove the namespace field from object to resolve.  "+
					"Object %s, Name=%q is declared with namespace %s",
				o.Object.GetObjectKind().GroupVersionKind(),
				metaObj.GetName(),
				ns))
		} else {
			node := v.nodes[len(v.nodes)-1]
			if node.Type == ast.Policyspace {
				v.errs.Add(errors.Errorf(
					"Objects declared in policyspace directories are not allowed to have a namespace specified, "+
						"remove the namespace field from object to resolve.  "+
						"Directory %q has declaration for %s, Name=%q with namespace %s",
					o.Object.GetObjectKind().GroupVersionKind(),
					metaObj.GetName(),
					ns))
			}
		}
	}
	return nil
}
