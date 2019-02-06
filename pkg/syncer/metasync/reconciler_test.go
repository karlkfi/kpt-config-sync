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
	nomosv1alpha1 "github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	syncerclient "github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/labeling"
	syncertesting "github.com/google/nomos/pkg/syncer/testing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type updateList struct {
	update nomosv1alpha1.Sync
	list   unstructured.UnstructuredList
}

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name              string
		actualSyncs       nomosv1alpha1.SyncList
		wantStatusUpdates []nomosv1alpha1.Sync
		wantUpdateList    []updateList
	}{
		{
			name: "update state for one sync",
			actualSyncs: nomosv1alpha1.SyncList{
				Items: []nomosv1alpha1.Sync{
					sync("", "v1", "Deployment", ""),
				},
			},
			wantStatusUpdates: []nomosv1alpha1.Sync{
				sync("", "v1", "Deployment", nomosv1alpha1.Syncing),
			},
		},
		{
			name: "update state for multiple syncs",
			actualSyncs: nomosv1alpha1.SyncList{
				Items: []nomosv1alpha1.Sync{
					sync("rbac.authorization.k8s.io", "v1", "Role", ""),
					sync("", "v1", "Deployment", ""),
					sync("", "v1", "ConfigMap", ""),
				},
			},
			wantStatusUpdates: []nomosv1alpha1.Sync{
				sync("rbac.authorization.k8s.io", "v1", "Role", nomosv1alpha1.Syncing),
				sync("", "v1", "Deployment", nomosv1alpha1.Syncing),
				sync("", "v1", "ConfigMap", nomosv1alpha1.Syncing),
			},
		},
		{
			name: "don't update state for one sync when unnecessary",
			actualSyncs: nomosv1alpha1.SyncList{
				Items: []nomosv1alpha1.Sync{
					sync("", "v1", "Deployment", nomosv1alpha1.Syncing),
				},
			},
		},
		{
			name: "don't update state for multiple syncs when unnecessary",
			actualSyncs: nomosv1alpha1.SyncList{
				Items: []nomosv1alpha1.Sync{
					sync("rbac.authorization.k8s.io", "v1", "Role", nomosv1alpha1.Syncing),
					sync("", "v1", "Deployment", nomosv1alpha1.Syncing),
					sync("", "v1", "ConfigMap", nomosv1alpha1.Syncing),
				},
			},
		},
		{
			name: "only update syncs with state change",
			actualSyncs: nomosv1alpha1.SyncList{
				Items: []nomosv1alpha1.Sync{
					sync("", "v1", "Secret", nomosv1alpha1.Syncing),
					sync("", "v1", "Service", nomosv1alpha1.Syncing),
					sync("", "v1", "Deployment", ""),
				},
			},
			wantStatusUpdates: []nomosv1alpha1.Sync{
				sync("", "v1", "Deployment", nomosv1alpha1.Syncing),
			},
		},
		{
			name: "finalize sync that is pending delete",
			actualSyncs: nomosv1alpha1.SyncList{
				Items: []nomosv1alpha1.Sync{
					withDeleteTimestamp(withFinalizer(sync("", "v1", "Deployment", nomosv1alpha1.Syncing))),
				},
			},
			wantUpdateList: []updateList{
				{
					update: withDeleteTimestamp(sync("", "v1", "Deployment", nomosv1alpha1.Syncing)),
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
			mockManager := syncertesting.NewMockRestartableManager(mockCtrl)

			testReconciler := &MetaReconciler{
				client:                 syncerclient.New(mockClient),
				cache:                  mockCache,
				genericResourceManager: mockManager,
			}

			mockCache.EXPECT().
				List(gomock.Any(), gomock.Any(), gomock.Any()).
				SetArg(2, tc.actualSyncs)

			mockManager.EXPECT().UpdateSyncResources(gomock.Any(), gomock.Any())
			for _, wantUpdateListDelete := range tc.wantUpdateList {
				// Updates involve first getting the resource from API Server.
				mockClient.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any())
				mockClient.EXPECT().
					Update(gomock.Any(), gomock.Eq(&wantUpdateListDelete.update))

				managed := labels.SelectorFromSet(labels.Set{labeling.ResourceManagementKey: labeling.Enabled})
				listOptions := &client.ListOptions{LabelSelector: managed}
				mockClient.EXPECT().
					List(gomock.Any(), gomock.Eq(listOptions), gomock.Eq(&wantUpdateListDelete.list))
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

func sync(group, version, kind string, state nomosv1alpha1.SyncState) nomosv1alpha1.Sync {
	s := nomosv1alpha1.Sync{
		ObjectMeta: metav1.ObjectMeta{
			Finalizers: []string{},
		},
		Spec: nomosv1alpha1.SyncSpec{
			Groups: []nomosv1alpha1.SyncGroup{
				{
					Group: group,
					Kinds: []nomosv1alpha1.SyncKind{
						{
							Kind: kind,
							Versions: []nomosv1alpha1.SyncVersion{
								{
									Version: version,
								},
							},
						},
					},
				},
			},
		},
	}
	if state != "" {
		s.Status = nomosv1alpha1.SyncStatus{
			GroupVersionKinds: []nomosv1alpha1.SyncGroupVersionKindStatus{
				{
					Group:   group,
					Version: version,
					Kind:    kind,
					Status:  state,
				},
			},
		}
	}
	return s
}

func withFinalizer(sync nomosv1alpha1.Sync) nomosv1alpha1.Sync {
	sync.SetFinalizers([]string{nomosv1alpha1.SyncFinalizer})
	return sync
}

func withDeleteTimestamp(sync nomosv1alpha1.Sync) nomosv1alpha1.Sync {
	t := metav1.NewTime(time.Unix(0, 0))
	sync.SetDeletionTimestamp(&t)
	return sync
}

func unstructuredList(gvk schema.GroupVersionKind) unstructured.UnstructuredList {
	ul := unstructured.UnstructuredList{}
	ul.SetGroupVersionKind(gvk)
	return ul
}
