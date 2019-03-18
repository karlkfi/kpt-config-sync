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

package namespaceconfig

import (
	"github.com/pkg/errors"

	listersv1 "github.com/google/nomos/clientgen/listers/policyhierarchy/v1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// ListPolicies returns all policies from API server.
func ListPolicies(namespaceConfigLister listersv1.NamespaceConfigLister,
	clusterConfigLister listersv1.ClusterConfigLister,
	syncLister listersv1.SyncLister,
	repoLister listersv1.RepoLister) (*AllPolicies, error) {
	policies := AllPolicies{
		NamespaceConfigs: make(map[string]v1.NamespaceConfig),
	}

	// NamespaceConfigs
	pn, err := namespaceConfigLister.List(labels.Everything())
	if err != nil {
		return nil, errors.Wrap(err, "failed to list NamespaceConfigs")
	}
	for _, n := range pn {
		policies.NamespaceConfigs[n.Name] = *n.DeepCopy()
	}

	// ClusterConfig
	cp, err := clusterConfigLister.List(labels.Everything())
	if err != nil {
		return nil, errors.Wrap(err, "failed to list ClusterConfigs")
	}

	if len(cp) > 1 {
		var names []string
		for _, c := range cp {
			names = append(names, c.Name)
		}
		return nil, errors.Errorf("found more than one ClusterConfig object. The cluster may be in an inconsistent state: %v", names)
	}
	if len(cp) == 1 {
		if cp[0].Name != v1.ClusterConfigName {
			return nil, errors.Errorf("expected ClusterConfig with name %q instead found %q", v1.ClusterConfigName, cp[0].Name)
		}
		policies.ClusterConfig = cp[0].DeepCopy()
	}

	// Syncs
	syncs, err := syncLister.List(labels.Everything())
	if err != nil {
		return nil, errors.Wrap(err, "failed to list Syncs")
	}
	if len(syncs) > 0 {
		policies.Syncs = make(map[string]v1.Sync)
	}
	for _, s := range syncs {
		policies.Syncs[s.Name] = *s.DeepCopy()
	}

	// Repo
	repos, err := repoLister.List(labels.Everything())
	if err != nil {
		return nil, errors.Wrap(err, "failed to list Repos")
	}

	if len(repos) > 1 {
		var names []string
		for _, r := range repos {
			names = append(names, r.Name)
		}
		return nil, errors.Errorf("found more than one Repo object. The cluster may be in an inconsistent state: %v", names)
	}
	if len(repos) == 1 {
		policies.Repo = repos[0].DeepCopy()
	}

	return &policies, nil
}
