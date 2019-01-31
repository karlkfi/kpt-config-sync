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
	"github.com/pkg/errors"

	listersv1 "github.com/google/nomos/clientgen/listers/policyhierarchy/v1"
	listersv1alpha1 "github.com/google/nomos/clientgen/listers/policyhierarchy/v1alpha1"
	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"k8s.io/apimachinery/pkg/labels"
)

// ListPolicies returns all policies from API server.
func ListPolicies(policyNodeLister listersv1.PolicyNodeLister,
	clusterPolicyLister listersv1.ClusterPolicyLister,
	syncLister listersv1alpha1.SyncLister) (*v1.AllPolicies, error) {
	policies := v1.AllPolicies{
		PolicyNodes: make(map[string]v1.PolicyNode),
	}

	pn, err := policyNodeLister.List(labels.Everything())
	if err != nil {
		return nil, errors.Wrap(err, "failed to list PolicyNodes")
	}
	for _, n := range pn {
		policies.PolicyNodes[n.Name] = *n.DeepCopy()
	}

	ls := labels.Everything()
	cp, err := clusterPolicyLister.List(ls)
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
		if cp[0].Name != v1.ClusterPolicyName {
			return nil, errors.Errorf("expected ClusterPolicy with name %q instead found %q", v1.ClusterPolicyName, cp[0].Name)
		}
		policies.ClusterPolicy = cp[0].DeepCopy()
	}

	syncs, err := syncLister.List(labels.Everything())
	if err != nil {
		return nil, errors.Wrap(err, "failed to list Syncs")
	}
	if len(syncs) > 0 {
		policies.Syncs = make(map[string]v1alpha1.Sync)
	}
	for _, s := range syncs {
		policies.Syncs[s.Name] = *s.DeepCopy()
	}

	return &policies, nil
}
