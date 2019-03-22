/*
Copyright 2017 The CSP Config Management Authors.
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
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/discovery"
)

// Scope runs after all transforms have completed.  This will verify that the final state of
// the tree meets various conditions before we set it on the API server.
type Scope struct {
	*visitor.Base
	errs    status.ErrorBuilder
	apiInfo *discovery.APIInfo
}

var _ ast.Visitor = &Scope{}

// NewScope returns a validator that checks if objects are in the correct scope in terms of namespace
// vs cluster.
// resourceLists is the list of supported types from the discovery client.
func NewScope() *Scope {
	pv := &Scope{
		Base: visitor.NewBase(),
	}
	pv.SetImpl(pv)
	return pv
}

// Error returns any errors encountered during processing
func (p *Scope) Error() *status.MultiError {
	return p.errs.Build()
}

// VisitRoot implement ast.Visitor.
func (p *Scope) VisitRoot(r *ast.Root) *ast.Root {
	p.apiInfo = discovery.GetAPIInfo(r)
	return p.Base.VisitRoot(r)
}

// VisitClusterObject implements Visitor
func (p *Scope) VisitClusterObject(o *ast.ClusterObject) *ast.ClusterObject {
	gvk := o.Object.GetObjectKind().GroupVersionKind()

	switch p.apiInfo.GetScope(gvk) {
	case discovery.NamespaceScope:
		p.errs.Add(vet.IllegalKindInClusterError{Resource: o})
	case discovery.UnknownScope:
		p.errs.Add(vet.UnknownObjectError{Resource: &o.FileObject})
	}

	return o
}

// VisitObject implements Visitor
func (p *Scope) VisitObject(o *ast.NamespaceObject) *ast.NamespaceObject {
	gvk := o.Object.GetObjectKind().GroupVersionKind()

	switch p.apiInfo.GetScope(gvk) {
	case discovery.ClusterScope:
		p.errs.Add(vet.IllegalKindInNamespacesError{Resource: o})
	case discovery.UnknownScope:
		p.errs.Add(vet.UnknownObjectError{Resource: &o.FileObject})
	}

	return o
}
