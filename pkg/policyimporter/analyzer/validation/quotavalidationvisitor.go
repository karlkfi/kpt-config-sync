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
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/util/multierror"
	corev1 "k8s.io/api/core/v1"
)

// QuotaValidator checks that ResourceQuota doesn't set scope related fields.
type QuotaValidator struct {
	*visitor.Base
	errs multierror.Builder
}

var _ ast.Visitor = &QuotaValidator{}

// NewQuotaValidator creates a new validator.
func NewQuotaValidator() *QuotaValidator {
	v := &QuotaValidator{
		Base: visitor.NewBase(),
	}
	v.Base.SetImpl(v)

	return v
}

// Error returns any errors encountered during processing
func (v *QuotaValidator) Error() error {
	return v.errs.Build()
}

// VisitObject implements Visitor
func (v *QuotaValidator) VisitObject(o *ast.NamespaceObject) *ast.NamespaceObject {
	if o.GetObjectKind().GroupVersionKind() == kinds.ResourceQuota() {
		quota := *o.FileObject.Object.(*corev1.ResourceQuota)
		// Scope-related fields aren't supported by the merge so error pre-emptively if set.
		if quota.Spec.Scopes != nil {
			v.errs.Add(vet.IllegalResourceQuotaFieldError{
				Path:          o.FileObject.Relative,
				ResourceQuota: quota,
				Field:         "scopes"})
		}
		if quota.Spec.ScopeSelector != nil {
			v.errs.Add(vet.IllegalResourceQuotaFieldError{
				Path:          o.FileObject.Relative,
				ResourceQuota: quota,
				Field:         "scopeSelector"})
		}
	}

	return v.Base.VisitObject(o)
}
