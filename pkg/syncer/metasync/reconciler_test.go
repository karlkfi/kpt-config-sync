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

package metasync

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	syncerclient "github.com/google/nomos/pkg/syncer/client"
	syncertesting "github.com/google/nomos/pkg/syncer/testing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type updateList struct {
	update v1alpha1.Sync
	list   unstructured.UnstructuredList
}

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name              string
		actualSyncs       v1alpha1.SyncList
		wantStatusUpdates []v1alpha1.Sync
		wantUpdateList    []updateList
	}{
		{
			name: "update state for one sync",
			actualSyncs: v1alpha1.SyncList{
				Items: []v1alpha1.Sync{
					makeSync("", "Deployment", ""),
				},
			},
			wantStatusUpdates: []v1alpha1.Sync{
				makeSync("", "Deployment", v1.Syncing),
			},
		},
		{
			name: "update state for multiple syncs",
			actualSyncs: v1alpha1.SyncList{
				Items: []v1alpha1.Sync{
					makeSync("rbac.authorization.k8s.io", "Role", ""),
					makeSync("", "Deployment", ""),
					makeSync("", "ConfigMap", ""),
				},
			},
			wantStatusUpdates: []v1alpha1.Sync{
				makeSync("rbac.authorization.k8s.io", "Role", v1.Syncing),
				makeSync("", "Deployment", v1.Syncing),
				makeSync("", "ConfigMap", v1.Syncing),
			},
		},
		{
			name: "don't update state for one sync when unnecessary",
			actualSyncs: v1alpha1.SyncList{
				Items: []v1alpha1.Sync{
					makeSync("", "Deployment", v1.Syncing),
				},
			},
		},
		{
			name: "don't update state for multiple syncs when unnecessary",
			actualSyncs: v1alpha1.SyncList{
				Items: []v1alpha1.Sync{
					makeSync("rbac.authorization.k8s.io", "Role", v1.Syncing),
					makeSync("", "Deployment", v1.Syncing),
					makeSync("", "ConfigMap", v1.Syncing),
				},
			},
		},
		{
			name: "only update syncs with state change",
			actualSyncs: v1alpha1.SyncList{
				Items: []v1alpha1.Sync{
					makeSync("", "Secret", v1.Syncing),
					makeSync("", "Service", v1.Syncing),
					makeSync("", "Deployment", ""),
				},
			},
			wantStatusUpdates: []v1alpha1.Sync{
				makeSync("", "Deployment", v1.Syncing),
			},
		},
		{
			name: "finalize sync that is pending delete",
			actualSyncs: v1alpha1.SyncList{
				Items: []v1alpha1.Sync{
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockClient := syncertesting.NewMockClient(mockCtrl)
			mockStatusClient := syncertesting.NewMockStatusWriter(mockCtrl)
			mockCache := syncertesting.NewMockCache(mockCtrl)
			mockDiscovery := syncertesting.NewMockDiscoveryInterface(mockCtrl)
			mockManager := syncertesting.NewMockRestartableManager(mockCtrl)

			testReconciler := &MetaReconciler{
				client:                 syncerclient.New(mockClient),
				cache:                  mockCache,
				discoveryClient:        mockDiscovery,
				genericResourceManager: mockManager,
				clientFactory: func() (client.Client, error) {
					return mockClient, nil
				},
			}

			mockCache.EXPECT().
				List(gomock.Any(), gomock.Any(), gomock.Any()).
				SetArg(2, tc.actualSyncs)

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

			mockManager.EXPECT().UpdateSyncResources(gomock.Any(), gomock.Any(), gomock.Any())
			for _, wantUpdateListDelete := range tc.wantUpdateList {
				// Updates involve first getting the resource from API Server.
				mockClient.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any())
				mockClient.EXPECT().
					Update(gomock.Any(), gomock.Eq(&wantUpdateListDelete.update))

				mockClient.EXPECT().
					List(gomock.Any(), gomock.Eq(&client.ListOptions{}), gomock.Eq(&wantUpdateListDelete.list))
			}

			mockClient.EXPECT().Status().Times(len(tc.wantStatusUpdates)).Return(mockStatusClient)
			for i := range tc.wantStatusUpdates {
				// Updates involve first getting the resource from API Server.
				mockClient.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any())
				mockStatusClient.EXPECT().
					Update(gomock.Any(), gomock.Eq(&tc.wantStatusUpdates[i]))
			}

			_, err := testReconciler.Reconcile(reconcile.Request{})
			if err != nil {
				t.Errorf("unexpected reconciliation error: %v", err)
			}
		})
	}
}

func makeSync(group, kind string, state v1.SyncState) v1alpha1.Sync {
	s := *v1alpha1.NewSync(group, kind)
	if state != "" {
		s.Status = v1alpha1.SyncStatus{Status: state}
	}
	return s
}

func withFinalizer(sync v1alpha1.Sync) v1alpha1.Sync {
	sync.SetFinalizers([]string{v1.SyncFinalizer})
	return sync
}

func withDeleteTimestamp(sync v1alpha1.Sync) v1alpha1.Sync {
	t := metav1.NewTime(time.Unix(0, 0))
	sync.SetDeletionTimestamp(&t)
	return sync
}

func unstructuredList(gvk schema.GroupVersionKind) unstructured.UnstructuredList {
	ul := unstructured.UnstructuredList{}
	ul.SetGroupVersionKind(gvk)
	return ul
}
