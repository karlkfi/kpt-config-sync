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

	listers_v1 "github.com/google/nomos/clientgen/listers/policyhierarchy/v1"
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// NewPolicyNode creates a PolicyNode from the given spec and name.
func NewPolicyNode(name string, spec *policyhierarchy_v1.PolicyNodeSpec) *policyhierarchy_v1.PolicyNode {
	return &policyhierarchy_v1.PolicyNode{
		TypeMeta: meta_v1.TypeMeta{
			Kind:       "PolicyNode",
			APIVersion: policyhierarchy_v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: meta_v1.ObjectMeta{
			Name: name,
		},
		Spec: *spec,
	}
}

// NewClusterPolicy creates a PolicyNode from the given spec and name.
func NewClusterPolicy(name string, spec *policyhierarchy_v1.ClusterPolicySpec) *policyhierarchy_v1.ClusterPolicy {
	return &policyhierarchy_v1.ClusterPolicy{
		TypeMeta: meta_v1.TypeMeta{
			Kind:       "ClusterPolicy",
			APIVersion: policyhierarchy_v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: meta_v1.ObjectMeta{
			Name: name,
		},
		Spec: *spec,
	}
}

// GetResourceVersion parses the resource version string into an int64
func GetResourceVersion(node *policyhierarchy_v1.PolicyNode) (int64, error) {
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
func ListPolicies(policyNodeLister listers_v1.PolicyNodeLister, clusterPolicyLister listers_v1.ClusterPolicyLister) (*policyhierarchy_v1.AllPolicies, error) {
	policies := policyhierarchy_v1.AllPolicies{
		PolicyNodes: make(map[string]policyhierarchy_v1.PolicyNode),
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
		if cp[0].Name != policyhierarchy_v1.ClusterPolicyName {
			return nil, errors.Errorf("expected ClusterPolicy with name %q instead found %q", policyhierarchy_v1.ClusterPolicyName, cp[0].Name)
		}
		policies.ClusterPolicy = cp[0].DeepCopy()
	}

	return &policies, nil
}
