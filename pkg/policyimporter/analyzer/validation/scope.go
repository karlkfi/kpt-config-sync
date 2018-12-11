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
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/policyimporter/meta"
	"github.com/google/nomos/pkg/util/multierror"
	"github.com/pkg/errors"
)

// Scope runs after all transforms have completed.  This will verify that the final state of
// the tree meets various conditions before we set it on the API server.
type Scope struct {
	*visitor.Base
	errs    multierror.Builder
	apiInfo *meta.APIInfo
}

// NewScope returns a validator that checks if objects are in the correct scope in terms of namespace
// vs cluster.
// resourceLists is the list of supported types from the discovery client.
func NewScope(apiInfo *meta.APIInfo) *Scope {
	pv := &Scope{
		Base:    visitor.NewBase(),
		apiInfo: apiInfo,
	}
	pv.SetImpl(pv)
	return pv
}

// Error returns any errors encountered during processing
func (p *Scope) Error() error {
	return p.errs.Build()
}

// VisitClusterObject implements Visitor
func (p *Scope) VisitClusterObject(o *ast.ClusterObject) *ast.ClusterObject {
	gvk := o.Object.GetObjectKind().GroupVersionKind()

	switch p.apiInfo.GetScope(gvk) {
	case meta.Namespace:
		p.errs.Add(errors.Errorf(
			"Namespace scoped object %s with Name %q in file %q cannot be declared in %q "+
				"directory.  Move declaration to the appropriate %q directory.",
			gvk,
			o.ToMeta().GetName(),
			o.Source,
			repo.ClusterDir,
			repo.NamespacesDir,
		))
	case meta.NotFound:
		p.errs.Add(vet.UnknownObjectError{FileObject: &o.FileObject})
	}

	return o
}

// VisitObject implements Visitor
func (p *Scope) VisitObject(o *ast.NamespaceObject) *ast.NamespaceObject {
	gvk := o.Object.GetObjectKind().GroupVersionKind()

	switch p.apiInfo.GetScope(gvk) {
	case meta.Cluster:
		p.errs.Add(errors.Errorf(
			"Cluster scoped object %s with Name %q from file %s cannot be declared inside "+
				"%q directory.  Move declaration to the %q directory.",
			gvk,
			o.ToMeta().GetName(),
			o.Source,
			repo.NamespacesDir,
			repo.ClusterDir,
		))
	case meta.NotFound:
		p.errs.Add(vet.UnknownObjectError{FileObject: &o.FileObject})
	}

	return o
}
