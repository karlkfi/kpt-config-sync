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

// Package differ contains code for comparing sync-enabled resources, not
// necessarily known at compile time.
package differ

import (
	"fmt"
	"reflect"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/util/meta"
	"github.com/pkg/errors"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Differ compares sync-enabled resources.
type Differ struct {
	// compareFields contains a list of fields to compare for the corresponding GroupVersionKind.
	compareFields map[schema.GroupVersionKind][]string
	// ignoreLabels is a list of labels to ignore when comparing resources.
	ignoreLabels []string
}

// NewDiffer returns a new Differ.
func NewDiffer(syncs []v1alpha1.Sync, ignoreLabels ...string) *Differ {
	compareFields := make(map[schema.GroupVersionKind][]string)
	for _, s := range syncs {
		for _, g := range s.Spec.Groups {
			for _, k := range g.Kinds {
				for _, v := range k.Versions {
					gvk := schema.GroupVersionKind{Group: g.Group, Version: v.Version, Kind: k.Kind}
					if len(v.CompareFields) == 0 {
						compareFields[gvk] = []string{"spec"}
					} else {
						compareFields[gvk] = v.CompareFields
					}
				}
			}
		}
	}

	return &Differ{
		compareFields: compareFields,
		ignoreLabels:  ignoreLabels,
	}
}

// Equal returns true if the two resources are equivalent according the fields specified in the corresponding Sync,
// their labels and their annotations.
func (d *Differ) Equal(lhs *unstructured.Unstructured, rhs *unstructured.Unstructured) bool {
	gvk := lhs.GroupVersionKind()
	if rgvk := rhs.GroupVersionKind(); gvk != rgvk {
		panic(fmt.Errorf("programmatic error: comparing two resources of different group, version, kinds: %s vs %s",
			gvk, rgvk))
	}
	// When checking for equality, we ignore the management label.
	// It's during reconciliation that we deal with mismatched management labels.
	lhs, rhs = d.copyWithoutIgnoredLabels(lhs), d.copyWithoutIgnoredLabels(rhs)
	if gvk == meta.ClusterRole {
		// We need special equality handling for ClusterRoles. Since depending on whether or not AggregationRules are set,
		// we will ignore the Rules field during comparison.
		return clusterRolesEqual(lhs, rhs) && action.MetaEqual(lhs, rhs)
	}
	for _, field := range d.compareFields[gvk] {
		lv, lok, lerr := unstructured.NestedFieldCopy(lhs.UnstructuredContent(), field)
		if lerr != nil {
			panic(errors.Wrapf(lerr, "programmatic error: comparing %s unstructured resource w/ unsupported fields", gvk))
		}
		rv, rok, rerr := unstructured.NestedFieldCopy(rhs.UnstructuredContent(), field)
		if rerr != nil {
			panic(errors.Wrapf(rerr, "programmatic error: comparing %s unstructured resource w/ unsupported fields", gvk))
		}
		if lok != rok {
			// missing field from one of the objects
			return false
		}
		if !reflect.DeepEqual(lv, rv) {
			// fields don't match
			return false
		}
	}
	return action.MetaEqual(lhs, rhs)
}

// copyWithoutIgnoredLabels returns a copy of the object without the Nomos management label.
func (d *Differ) copyWithoutIgnoredLabels(obj *unstructured.Unstructured) *unstructured.Unstructured {
	objCopy := obj.DeepCopy()
	labels := objCopy.GetLabels()
	for _, l := range d.ignoreLabels {
		delete(labels, l)
	}
	objCopy.SetLabels(labels)
	return objCopy
}

// clusterRolesEqual returns true if the two ClusterRoles are equivalent.
func clusterRolesEqual(lhs *unstructured.Unstructured, rhs *unstructured.Unstructured) bool {
	lcp, rcp := clusterRoleOrDie(lhs), clusterRoleOrDie(rhs)
	if lcp.AggregationRule != nil || rcp.AggregationRule != nil {
		return reflect.DeepEqual(lcp.AggregationRule, rcp.AggregationRule)
	}
	return reflect.DeepEqual(lcp.Rules, rcp.Rules)
}

// clusterRoleOrDie returns a ClusterRole from Unstructured, otherwise it panics.
func clusterRoleOrDie(obj *unstructured.Unstructured) *rbacv1.ClusterRole {
	cr := &rbacv1.ClusterRole{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), cr); err != nil {
		panic(errors.Wrap(err, "programmatic error: comparing ClusterRole that cannot be converted to its type"))
	}
	return cr
}
