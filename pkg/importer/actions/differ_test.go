/*
Copyright 2017 The Kubernetes Authors.
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

package actions

import (
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"k8s.io/api/policy/v1beta1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type testCase struct {
	testName                           string
	oldNodes, newNodes                 []v1.NamespaceConfig
	oldClusterConfig, newClusterConfig *v1.ClusterConfig
	oldSyncs, newSyncs                 []v1.Sync
	// String representation of expected actions
	expected []string
}

func TestDiffer(t *testing.T) {
	for _, test := range []testCase{
		{
			testName: "Nil",
		},
		{
			testName: "All Empty",
			oldNodes: []v1.NamespaceConfig{},
			newNodes: []v1.NamespaceConfig{},
		},
		{
			testName: "One node Create",
			oldNodes: []v1.NamespaceConfig{},
			newNodes: []v1.NamespaceConfig{
				namespaceConfig("r"),
			},
			expected: []string{"configmanagement.gke.io/v1/NamespaceConfigs/r/create"},
		},
		{
			testName: "One node delete",
			oldNodes: []v1.NamespaceConfig{
				namespaceConfig("r"),
			},
			newNodes: []v1.NamespaceConfig{},
			expected: []string{"configmanagement.gke.io/v1/NamespaceConfigs/r/update"},
		},
		{
			testName: "Rename root node",
			oldNodes: []v1.NamespaceConfig{
				namespaceConfig("r"),
			},
			newNodes: []v1.NamespaceConfig{
				namespaceConfig("r2"),
			},
			expected: []string{
				"configmanagement.gke.io/v1/NamespaceConfigs/r2/create",
				"configmanagement.gke.io/v1/NamespaceConfigs/r/update",
			},
		},
		{
			testName: "Create 2 nodes",
			oldNodes: []v1.NamespaceConfig{
				namespaceConfig("r"),
			},
			newNodes: []v1.NamespaceConfig{
				namespaceConfig("r"),
				namespaceConfig("c1"),
				namespaceConfig("c2"),
			},
			expected: []string{
				"configmanagement.gke.io/v1/NamespaceConfigs/c1/create",
				"configmanagement.gke.io/v1/NamespaceConfigs/c2/create",
			},
		},
		{
			testName: "Create 2 nodes and delete 2",
			oldNodes: []v1.NamespaceConfig{
				namespaceConfig("r"),
				namespaceConfig("co1"),
				namespaceConfig("co2"),
			},
			newNodes: []v1.NamespaceConfig{
				namespaceConfig("r"),
				namespaceConfig("c2"),
				namespaceConfig("c1"),
			},
			expected: []string{
				"configmanagement.gke.io/v1/NamespaceConfigs/c1/create",
				"configmanagement.gke.io/v1/NamespaceConfigs/c2/create",
				"configmanagement.gke.io/v1/NamespaceConfigs/co2/update",
				"configmanagement.gke.io/v1/NamespaceConfigs/co1/update",
			},
		},
		{
			testName:         "ClusterConfig create",
			newClusterConfig: clusterConfig("foo", true),
			expected: []string{
				"configmanagement.gke.io/v1/ClusterConfigs/foo/create",
			},
		},
		{
			testName:         "ClusterConfig update",
			oldClusterConfig: clusterConfig("foo", true),
			newClusterConfig: clusterConfig("foo", false),
			expected: []string{
				"configmanagement.gke.io/v1/ClusterConfigs/foo/update",
			},
		},
		{
			testName:         "ClusterConfig update no change",
			oldClusterConfig: clusterConfig("foo", true),
			newClusterConfig: clusterConfig("foo", true),
		},
		{
			testName:         "ClusterConfig delete",
			oldClusterConfig: clusterConfig("foo", true),
			expected: []string{
				"configmanagement.gke.io/v1/ClusterConfigs/foo/delete",
			},
		},
		{
			testName: "Create 2 nodes and a ClusterConfig",
			oldNodes: []v1.NamespaceConfig{
				namespaceConfig("r"),
			},
			newNodes: []v1.NamespaceConfig{
				namespaceConfig("r"),
				namespaceConfig("c2"),
				namespaceConfig("c1"),
			},
			newClusterConfig: clusterConfig("foo", true),
			expected: []string{
				"configmanagement.gke.io/v1/NamespaceConfigs/c1/create",
				"configmanagement.gke.io/v1/NamespaceConfigs/c2/create",
				"configmanagement.gke.io/v1/ClusterConfigs/foo/create",
			},
		},
		{
			testName: "Empty Syncs",
			oldSyncs: []v1.Sync{},
			newSyncs: []v1.Sync{},
		},
		{
			testName: "Sync create",
			oldSyncs: []v1.Sync{},
			newSyncs: []v1.Sync{
				*v1.NewSync("", "ResourceQuota"),
			},
			expected: []string{
				"configmanagement.gke.io/v1/Syncs/resourcequota/create",
			},
		},
		{
			testName: "Sync update no change",
			oldSyncs: []v1.Sync{
				*v1.NewSync("", "ResourceQuota"),
			},
			newSyncs: []v1.Sync{
				*v1.NewSync("", "ResourceQuota"),
			},
		},
		{
			testName: "Sync delete",
			oldSyncs: []v1.Sync{
				*v1.NewSync("", "ResourceQuota"),
			},
			newSyncs: []v1.Sync{},
			expected: []string{
				"configmanagement.gke.io/v1/Syncs/resourcequota/delete",
			},
		},
	} {
		t.Run(test.testName, func(t *testing.T) {
			g := NewDiffer(NewFactories(nil, nil, nil, nil, nil))
			g.SortDiff = true

			gotActions := g.Diff(
				allConfigs(test.oldNodes, test.oldClusterConfig, test.oldSyncs),
				allConfigs(test.newNodes, test.newClusterConfig, test.newSyncs))

			if len(gotActions) != len(test.expected) {
				t.Fatalf("Actual number of actions was %d but expected %d",
					len(gotActions), len(test.expected))
			}

			var actual []string
			for _, a := range gotActions {
				actual = append(actual, a.String())
			}
			sort.Strings(test.expected)
			sort.Strings(actual)
			if !cmp.Equal(test.expected, actual) {
				t.Fatalf("Exepcted and actual actions differ: %s", cmp.Diff(test.expected, actual))
			}

			namespaceConfigs := make(map[string]v1.NamespaceConfig)
			for _, pn := range test.oldNodes {
				namespaceConfigs[pn.Name] = pn
			}
			for _, a := range gotActions {
				executeAction(t, a, namespaceConfigs)
			}
		})
	}
}

func executeAction(t *testing.T, a action.Interface, nodes map[string]v1.NamespaceConfig) {
	if a.Kind() != action.Plural(v1.NamespaceConfig{}) {
		// We only test transient states for NamespaceConfigs
		return
	}
	op := a.Operation()
	switch op {
	case action.CreateOperation, action.UpdateOperation, action.UpsertOperation:
		nodes[a.Name()] = namespaceConfig(a.Name())
	case action.DeleteOperation:
		delete(nodes, a.Name())
	default:
		t.Fatalf("Unexpected operation: %v", op)
	}
}

func namespaceConfig(name string) v1.NamespaceConfig {
	return v1.NamespaceConfig{
		ObjectMeta: meta.ObjectMeta{
			Name: name,
		},
		Spec: v1.NamespaceConfigSpec{},
	}
}

func clusterConfig(name string, priviledged bool) *v1.ClusterConfig {
	return &v1.ClusterConfig{
		ObjectMeta: meta.ObjectMeta{
			Name: name,
		},
		Spec: v1.ClusterConfigSpec{
			Resources: []v1.GenericResources{
				{
					Group: v1beta1.GroupName,
					Kind:  "PodSecurityPolicy",
					Versions: []v1.GenericVersionResources{
						{
							Version: v1beta1.SchemeGroupVersion.Version,
							Objects: []runtime.RawExtension{
								{
									Object: &v1beta1.PodSecurityPolicy{
										Spec: v1beta1.PodSecurityPolicySpec{Privileged: priviledged},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func allConfigs(nodes []v1.NamespaceConfig, clusterConfig *v1.ClusterConfig, syncs []v1.Sync) namespaceconfig.AllConfigs {
	configs := namespaceconfig.AllConfigs{
		ClusterConfig: clusterConfig,
	}

	for i, n := range nodes {
		if i == 0 {
			configs.NamespaceConfigs = make(map[string]v1.NamespaceConfig)
		}
		configs.NamespaceConfigs[n.Name] = n
	}

	if len(syncs) > 0 {
		configs.Syncs = make(map[string]v1.Sync)
	}
	for _, s := range syncs {
		configs.Syncs[s.Name] = s
	}

	return configs
}
