package sync

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	syncerclient "github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/metrics"
	syncertesting "github.com/google/nomos/pkg/syncer/testing"
	"github.com/google/nomos/pkg/syncer/testing/mocks"
	utilmocks "github.com/google/nomos/pkg/util/testing/mocks"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type updateList struct {
	update v1.Sync
	list   unstructured.UnstructuredList
}

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name                 string
		actualSyncs          v1.SyncList
		reconcileRequestName string
		wantStatusUpdates    []v1.Sync
		wantUpdateList       []updateList
		wantForceRestart     bool
	}{
		{
			name: "update state for one sync",
			actualSyncs: v1.SyncList{
				Items: []v1.Sync{
					makeSync("", "Deployment", ""),
				},
			},
			wantStatusUpdates: []v1.Sync{
				makeSync("", "Deployment", v1.Syncing),
			},
		},
		{
			name: "update state for multiple syncs",
			actualSyncs: v1.SyncList{
				Items: []v1.Sync{
					makeSync("rbac.authorization.k8s.io", "Role", ""),
					makeSync("", "Deployment", ""),
					makeSync("", "ConfigMap", ""),
				},
			},
			wantStatusUpdates: []v1.Sync{
				makeSync("rbac.authorization.k8s.io", "Role", v1.Syncing),
				makeSync("", "Deployment", v1.Syncing),
				makeSync("", "ConfigMap", v1.Syncing),
			},
		},
		{
			name: "don't update state for one sync when unnecessary",
			actualSyncs: v1.SyncList{
				Items: []v1.Sync{
					makeSync("", "Deployment", v1.Syncing),
				},
			},
		},
		{
			name: "don't update state for multiple syncs when unnecessary",
			actualSyncs: v1.SyncList{
				Items: []v1.Sync{
					makeSync("rbac.authorization.k8s.io", "Role", v1.Syncing),
					makeSync("", "Deployment", v1.Syncing),
					makeSync("", "ConfigMap", v1.Syncing),
				},
			},
		},
		{
			name: "only update syncs with state change",
			actualSyncs: v1.SyncList{
				Items: []v1.Sync{
					makeSync("", "Secret", v1.Syncing),
					makeSync("", "Service", v1.Syncing),
					makeSync("", "Deployment", ""),
				},
			},
			wantStatusUpdates: []v1.Sync{
				makeSync("", "Deployment", v1.Syncing),
			},
		},
		{
			name: "finalize sync that is pending delete",
			actualSyncs: v1.SyncList{
				Items: []v1.Sync{
					withDeleteTimestamp(withFinalizer(makeSync("", "Deployment", v1.Syncing))),
				},
			},
			wantUpdateList: []updateList{
				{
					update: withDeleteTimestamp(makeSync("", "Deployment", v1.Syncing)),
					list:   unstructuredList(schema.GroupVersionKind{Version: "v1", Kind: "DeploymentList"}),
				},
			},
		},
		{
			name:                 "force restart reconcile request restarts SubManager",
			reconcileRequestName: ForceRestart,
			actualSyncs: v1.SyncList{
				Items: []v1.Sync{
					makeSync("", "Deployment", ""),
				},
			},
			wantStatusUpdates: []v1.Sync{
				makeSync("", "Deployment", v1.Syncing),
			},
			wantForceRestart: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockClient := mocks.NewMockClient(mockCtrl)
			mockStatusClient := mocks.NewMockStatusWriter(mockCtrl)
			mockCache := mocks.NewMockCache(mockCtrl)
			mockDiscovery := mocks.NewMockDiscoveryInterface(mockCtrl)
			mockManager := utilmocks.NewMockRestartableManager(mockCtrl)

			testReconciler := &MetaReconciler{
				client:          syncerclient.New(mockClient, metrics.APICallDuration),
				cache:           mockCache,
				discoveryClient: mockDiscovery,
				builder:         newSyncAwareBuilder(),
				subManager:      mockManager,
				clientFactory: func() (client.Client, error) {
					return mockClient, nil
				},
				now: syncertesting.Now,
			}

			mockCache.EXPECT().
				List(gomock.Any(), gomock.Any(), gomock.Any()).
				SetArg(1, tc.actualSyncs)

			mockDiscovery.EXPECT().
				ServerResources().Return(
				[]*metav1.APIResourceList{
					{
						GroupVersion: "/v1",
						APIResources: []metav1.APIResource{
							{
								Kind: "ConfigMap",
							},
							{
								Kind: "Deployment",
							},
							{
								Kind: "Secret",
							},
							{
								Kind: "Service",
							},
						},
					},
					{
						GroupVersion: "rbac.authorization.k8s.io/v1",
						APIResources: []metav1.APIResource{
							{
								Kind: "Role",
							},
						},
					},
					{
						GroupVersion: "rbac.authorization.k8s.io/v1beta1",
						APIResources: []metav1.APIResource{
							{
								Kind: "Role",
							},
						},
					},
				}, nil)

			mockManager.EXPECT().Restart(gomock.Any(), gomock.Eq(tc.wantForceRestart))
			for _, wantUpdateList := range tc.wantUpdateList {
				// Updates involve first getting the resource from API Server.
				mockClient.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any())
				mockClient.EXPECT().
					Update(gomock.Any(), gomock.Eq(&wantUpdateList.update))

				mockClient.EXPECT().
					List(gomock.Any(), gomock.Eq(&wantUpdateList.list), gomock.Eq(&client.ListOptions{}))
			}

			mockClient.EXPECT().Status().Times(len(tc.wantStatusUpdates)).Return(mockStatusClient)
			for i := range tc.wantStatusUpdates {
				// Updates involve first getting the resource from API Server.
				mockClient.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any())
				mockStatusClient.EXPECT().
					Update(gomock.Any(), gomock.Eq(&tc.wantStatusUpdates[i]))
			}

			_, err := testReconciler.Reconcile(reconcile.Request{
				NamespacedName: apimachinerytypes.NamespacedName{
					Name: tc.reconcileRequestName,
				},
			})
			if err != nil {
				t.Errorf("unexpected reconciliation error: %v", err)
			}
		})
	}
}

func makeSync(group, kind string, state v1.SyncState) v1.Sync {
	s := *v1.NewSync(schema.GroupKind{Group: group, Kind: kind})
	if state != "" {
		s.Status = v1.SyncStatus{Status: state}
	}
	return s
}

func withFinalizer(sync v1.Sync) v1.Sync {
	sync.SetFinalizers([]string{v1.SyncFinalizer})
	return sync
}

func withDeleteTimestamp(sync v1.Sync) v1.Sync {
	t := metav1.NewTime(time.Unix(0, 0))
	sync.SetDeletionTimestamp(&t)
	return sync
}

func unstructuredList(gvk schema.GroupVersionKind) unstructured.UnstructuredList {
	ul := unstructured.UnstructuredList{}
	ul.SetGroupVersionKind(gvk)
	return ul
}
