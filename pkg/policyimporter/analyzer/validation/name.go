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
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/util/multierror"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NameValidator is a validator that checks for resource name uniqueness.
type NameValidator struct {
	*visitor.Base
	errs        multierror.Builder
	nameChecker nameChecker
}

// NameValidator implements ast.CheckingVisitor
var _ ast.CheckingVisitor = &NameValidator{}

// NewNameValidator returns a new NameValidator validator.
func NewNameValidator() *NameValidator {
	n := &NameValidator{
		Base: visitor.NewBase(),
	}
	n.SetImpl(n)
	return n
}

// Error returns any errors encountered during processing
func (v *NameValidator) Error() error {
	return v.errs.Build()
}

// VisitClusterObjectList implements Visitor
func (v *NameValidator) VisitClusterObjectList(o ast.ClusterObjectList) ast.Node {
	v.nameChecker = nameChecker{}
	return v.Base.VisitClusterObjectList(o)
}

// VisitClusterObject implements Visitor
func (v *NameValidator) VisitClusterObject(o *ast.ClusterObject) ast.Node {
	v.handleFileObject("Cluster", &o.FileObject)
	return o
}

// VisitObjectList implements Visitor
func (v *NameValidator) VisitObjectList(o ast.ObjectList) ast.Node {
	v.nameChecker = nameChecker{}
	return v.Base.VisitObjectList(o)
}

// VisitObject implements Visitor
func (v *NameValidator) VisitObject(o *ast.NamespaceObject) ast.Node {
	v.handleFileObject("Namespace", &o.FileObject)
	return o
}

func (v *NameValidator) handleFileObject(scope string, o *ast.FileObject) {
	if err := v.nameChecker.add(scope, o); err != nil {
		v.errs.Add(err)
	}
}

// nameChecker is a map of GroupKind to map of object name to ast.FileObject which helps facilitate
// name uniqueness.
type nameChecker map[schema.GroupKind]map[string]*ast.FileObject

// add will add the object to nameChecker.  If an object of the same GroupKind and NameValidator does not
// exist, the object will be added and nil will be returned.  If an object of the same GroupKind and
// NameValidator already exists, the object will not be added and the existing object will be returned.
func (n nameChecker) add(scope string, o *ast.FileObject) error {
	gk := o.GetObjectKind().GroupVersionKind().GroupKind()
	gkObjs, found := n[gk]
	if !found {
		gkObjs = map[string]*ast.FileObject{}
		n[gk] = gkObjs
	}

	name := o.ToMeta().GetName()
	prev, found := gkObjs[name]
	if found {
		return errors.Errorf(
			"%s scoped object %s with Name %q has duplicate declarations:\n%s:\n%#v\n%s:\n%#v",
			scope,
			o.GetObjectKind().GroupVersionKind(),
			o.ToMeta().GetName(),
			prev.Source,
			prev.Object,
			o.Source,
			o.Object,
		)
	}
	gkObjs[name] = o
	return nil
}
