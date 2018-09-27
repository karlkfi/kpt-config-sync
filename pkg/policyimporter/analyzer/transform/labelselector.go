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

package transform

import (
	"encoding/json"
	"fmt"

	policyhierarchyv1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	policyhierarchyv1alpha1 "github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labelpkg "k8s.io/apimachinery/pkg/labels"
)

func isSelected(labels map[string]string, labelselector *metav1.LabelSelector) (bool, error) {
	selector, err := metav1.LabelSelectorAsSelector(labelselector)
	if err != nil {
		return false, fmt.Errorf("invalid label selector: %v", err)
	}
	if selector.Empty() {
		return false, fmt.Errorf("empty label selector")
	}

	return selector.Matches(labelpkg.Set(labels)), nil
}

// isPolicyApplicableToNamespace returns whether the NamespaceSelector annotation on the given policy object
// matches the given labels on a namespace.
// The policy is applicable if it has no such annotation.
func isPolicyApplicableToNamespace(namespaceLabels map[string]string, policy metav1.Object) bool {
	ls, exists := policy.GetAnnotations()[policyhierarchyv1.NamespaceSelectorAnnotationKey]
	if !exists {
		return true
	}
	var ns policyhierarchyv1alpha1.NamespaceSelector
	if err := json.Unmarshal([]byte(ls), &ns); err != nil {
		panic(errors.Wrapf(err, "failed to unmarshal NamespaceSelector in object %q", policy.GetName()))
	}
	selected, err := isSelected(namespaceLabels, &ns.Spec.Selector)
	if err != nil {
		panic(errors.Wrapf(err, "failed to evaluate LabelSelector in object %q", policy.GetName()))
	}
	return selected
}
