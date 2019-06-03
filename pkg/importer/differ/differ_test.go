package differ

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/metrics"
	"github.com/google/nomos/pkg/syncer/testing/mocks"

	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"k8s.io/api/policy/v1beta1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type testCase struct {
	testName                           string
	oldNodes, newNodes                 []*v1.NamespaceConfig
	oldClusterConfig, newClusterConfig *v1.ClusterConfig
	oldSyncs, newSyncs                 []*v1.Sync
	// String representation of expected actions
	wantCreate []runtime.Object
	wantUpdate []runtime.Object
	wantDelete []runtime.Object
}

func TestDiffer(t *testing.T) {
	for _, test := range []testCase{
		{
			testName: "Nil",
		},
		{
			testName: "All Empty",
			oldNodes: []*v1.NamespaceConfig{},
			newNodes: []*v1.NamespaceConfig{},
		},
		{
			testName: "One node Create",
			oldNodes: []*v1.NamespaceConfig{},
			newNodes: []*v1.NamespaceConfig{
				namespaceConfig("r"),
			},
			wantCreate: []runtime.Object{
				namespaceConfig("r"),
			},
		},
		{
			testName: "One node delete",
			oldNodes: []*v1.NamespaceConfig{
				namespaceConfig("r"),
			},
			newNodes: []*v1.NamespaceConfig{},
			wantDelete: []runtime.Object{
				namespaceConfig("r"),
			},
		},
		{
			testName: "Rename root node",
			oldNodes: []*v1.NamespaceConfig{
				namespaceConfig("r"),
			},
			newNodes: []*v1.NamespaceConfig{
				namespaceConfig("r2"),
			},
			wantDelete: []runtime.Object{
				namespaceConfig("r"),
			},
			wantCreate: []runtime.Object{
				namespaceConfig("r2"),
			},
		},
		{
			testName: "Create 2 nodes",
			oldNodes: []*v1.NamespaceConfig{
				namespaceConfig("r"),
			},
			newNodes: []*v1.NamespaceConfig{
				namespaceConfig("r"),
				namespaceConfig("c1"),
				namespaceConfig("c2"),
			},
			wantCreate: []runtime.Object{
				namespaceConfig("c1"),
				namespaceConfig("c2"),
			},
		},
		{
			testName: "Create 2 nodes and delete 2",
			oldNodes: []*v1.NamespaceConfig{
				namespaceConfig("r"),
				namespaceConfig("co1"),
				namespaceConfig("co2"),
			},
			newNodes: []*v1.NamespaceConfig{
				namespaceConfig("r"),
				namespaceConfig("c2"),
				namespaceConfig("c1"),
			},
			wantCreate: []runtime.Object{
				namespaceConfig("c1"),
				namespaceConfig("c2"),
			},
			wantDelete: []runtime.Object{
				namespaceConfig("co1"),
				namespaceConfig("co2"),
			},
		},
		{
			testName:         "ClusterConfig create",
			newClusterConfig: clusterConfig("foo", true),
			wantCreate: []runtime.Object{
				clusterConfig("foo", true),
			},
		},
		{
			testName:         "ClusterConfig update",
			oldClusterConfig: clusterConfig("foo", true),
			newClusterConfig: clusterConfig("foo", false),
			wantUpdate: []runtime.Object{
				clusterConfig("foo", false),
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
			wantDelete: []runtime.Object{
				clusterConfig("foo", true),
			},
		},
		{
			testName: "Create 2 nodes and a ClusterConfig",
			oldNodes: []*v1.NamespaceConfig{
				namespaceConfig("r"),
			},
			newNodes: []*v1.NamespaceConfig{
				namespaceConfig("r"),
				namespaceConfig("c2"),
				namespaceConfig("c1"),
			},
			newClusterConfig: clusterConfig("foo", true),
			wantCreate: []runtime.Object{
				clusterConfig("foo", true),
				namespaceConfig("c1"),
				namespaceConfig("c2"),
			},
		},
		{
			testName: "Empty Syncs",
			oldSyncs: []*v1.Sync{},
			newSyncs: []*v1.Sync{},
		},
		{
			testName: "Sync create",
			oldSyncs: []*v1.Sync{},
			newSyncs: []*v1.Sync{
				v1.NewSync("", "ResourceQuota"),
			},
			wantCreate: []runtime.Object{
				v1.NewSync("", "ResourceQuota"),
			},
		},
		{
			testName: "Sync update no change",
			oldSyncs: []*v1.Sync{
				v1.NewSync("", "ResourceQuota"),
			},
			newSyncs: []*v1.Sync{
				v1.NewSync("", "ResourceQuota"),
			},
		},
		{
			testName: "Sync delete",
			oldSyncs: []*v1.Sync{
				v1.NewSync("", "ResourceQuota"),
			},
			newSyncs: []*v1.Sync{},
			wantDelete: []runtime.Object{
				v1.NewSync("", "ResourceQuota"),
			},
		},
	} {
		t.Run(test.testName, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockClient := mocks.NewMockClient(mockCtrl)
			for _, c := range test.wantCreate {
				mockClient.EXPECT().Create(gomock.Any(), gomock.Eq(c))
			}
			for _, c := range test.wantDelete {
				mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any())
				mockClient.EXPECT().Delete(gomock.Any(), gomock.Eq(c), gomock.Any())
			}
			for _, c := range test.wantUpdate {
				mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any())
				mockClient.EXPECT().Update(gomock.Any(), gomock.Eq(c))
			}

			err := Update(context.Background(), client.New(mockClient, metrics.APICallDuration),
				allConfigs(test.oldNodes, test.oldClusterConfig, test.oldSyncs),
				allConfigs(test.newNodes, test.newClusterConfig, test.newSyncs))
			if err != nil {
				t.Fatalf("unexpected error in diff: %v", err)
			}
		})
	}
}

func namespaceConfig(name string) *v1.NamespaceConfig {
	return &v1.NamespaceConfig{
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

func allConfigs(nodes []*v1.NamespaceConfig, clusterConfig *v1.ClusterConfig, syncs []*v1.Sync) namespaceconfig.AllConfigs {
	configs := namespaceconfig.AllConfigs{
		ClusterConfig: clusterConfig,
	}

	for i, n := range nodes {
		if i == 0 {
			configs.NamespaceConfigs = make(map[string]v1.NamespaceConfig)
		}
		configs.NamespaceConfigs[n.Name] = *n
	}

	if len(syncs) > 0 {
		configs.Syncs = make(map[string]v1.Sync)
	}
	for _, s := range syncs {
		configs.Syncs[s.Name] = *s
	}

	return configs
}
