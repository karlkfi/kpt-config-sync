/*
Copyright 2018 The CSP Config Management Authors.

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

package selectors

import (
	"encoding/json"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IsPolicyApplicableToNamespace returns whether the NamespaceSelector
// annotation on the given policy object matches the given labels on a
// namespace.  The policy is applicable if it has no such annotation.
func IsPolicyApplicableToNamespace(namespaceLabels map[string]string, policy metav1.Object) (bool, status.Error) {
	ls, exists := policy.GetAnnotations()[v1.NamespaceSelectorAnnotationKey]
	if !exists {
		return true, nil
	}
	var ns v1.NamespaceSelector
	if err := json.Unmarshal([]byte(ls), &ns); err != nil {
		// TODO(b/122738890)
		return false, status.UndocumentedWrapf(err, "failed to unmarshal NamespaceSelector in object %q", policy.GetName())
	}
	selector, err := AsPopulatedSelector(&ns.Spec.Selector)
	if err != nil {
		return false, vet.InvalidSelectorError{Name: ns.Name, Cause: err}
	}
	return IsSelected(namespaceLabels, selector), nil
}
