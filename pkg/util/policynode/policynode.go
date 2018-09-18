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

package policynode

import (
	"strconv"

	"github.com/pkg/errors"

	listersv1 "github.com/google/nomos/clientgen/listers/policyhierarchy/v1"
	policyhierarchyv1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// NewPolicyNode creates a PolicyNode from the given spec and name.
func NewPolicyNode(name string, spec *policyhierarchyv1.PolicyNodeSpec) *policyhierarchyv1.PolicyNode {
	return &policyhierarchyv1.PolicyNode{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PolicyNode",
			APIVersion: policyhierarchyv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: *spec,
	}
}

// NewClusterPolicy creates a PolicyNode from the given spec and name.
func NewClusterPolicy(name string, spec *policyhierarchyv1.ClusterPolicySpec) *policyhierarchyv1.ClusterPolicy {
	return &policyhierarchyv1.ClusterPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterPolicy",
			APIVersion: policyhierarchyv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: *spec,
	}
}

// GetResourceVersion parses the resource version string into an int64
func GetResourceVersion(node *policyhierarchyv1.PolicyNode) (int64, error) {
	resourceVersionStr := node.ResourceVersion
	if resourceVersionStr == "" {
		return 0, errors.Errorf("Empty resource version in %#v", node)
	}

	resourceVersion, err := strconv.ParseInt(resourceVersionStr, 10, 64)
	if err != nil {
		return 0, errors.Wrapf(err, "Failed to parse resource version from %#v", node)
	}

	return resourceVersion, nil
}

// ListPolicies returns all policies from API server.
func ListPolicies(policyNodeLister listersv1.PolicyNodeLister, clusterPolicyLister listersv1.ClusterPolicyLister) (*policyhierarchyv1.AllPolicies, error) {
	policies := policyhierarchyv1.AllPolicies{
		PolicyNodes: make(map[string]policyhierarchyv1.PolicyNode),
	}

	pn, err := policyNodeLister.List(labels.Everything())
	if err != nil {
		return nil, errors.Wrap(err, "failed to list PolicyNodes")
	}
	for _, n := range pn {
		policies.PolicyNodes[n.Name] = *n.DeepCopy()
	}

	cp, err := clusterPolicyLister.List(labels.Everything())
	if err != nil {
		return nil, errors.Wrap(err, "failed to list ClusterPolicies")
	}

	if len(cp) > 1 {
		var names []string
		for _, c := range cp {
			names = append(names, c.Name)
		}
		return nil, errors.Errorf("found more than one ClusterPolicy object. The cluster may be in an inconsistent state: %v", names)
	}
	if len(cp) == 1 {
		if cp[0].Name != policyhierarchyv1.ClusterPolicyName {
			return nil, errors.Errorf("expected ClusterPolicy with name %q instead found %q", policyhierarchyv1.ClusterPolicyName, cp[0].Name)
		}
		policies.ClusterPolicy = cp[0].DeepCopy()
	}

	return &policies, nil
}
