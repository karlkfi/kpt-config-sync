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

package namespaceconfig

import (
	listersv1 "github.com/google/nomos/clientgen/listers/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/labels"
)

// ListPolicies returns all policies from API server.
func ListPolicies(namespaceConfigLister listersv1.NamespaceConfigLister,
	clusterConfigLister listersv1.ClusterConfigLister,
	syncLister listersv1.SyncLister) (*AllPolicies, error) {
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
	policies.Syncs, err = ListSyncs(syncLister)
	return &policies, err
}

// ListSyncs gets a map-by-name of Syncs currently present in the cluster from
// the provided lister.
func ListSyncs(syncLister listersv1.SyncLister) (ret map[string]v1.Sync, err error) {
	syncs, err := syncLister.List(labels.Everything())
	if err != nil {
		return nil, errors.Wrap(err, "failed to list Syncs")
	}
	if len(syncs) > 0 {
		ret = make(map[string]v1.Sync, len(syncs))
	}
	for _, s := range syncs {
		ret[s.Name] = *s.DeepCopy()
	}
	return ret, err
}
