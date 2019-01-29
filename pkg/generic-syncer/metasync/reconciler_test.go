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
	"github.com/google/nomos/pkg/generic-syncer/client"
	syncertesting "github.com/google/nomos/pkg/generic-syncer/testing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name              string
		actualSyncs       nomosv1alpha1.SyncList
		wantUpdates       []nomosv1alpha1.Sync
		wantStatusUpdates []nomosv1alpha1.Sync
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
			name: "update state for multiple Group/Kinds in one sync",
			actualSyncs: nomosv1alpha1.SyncList{
				Items: []nomosv1alpha1.Sync{
					{
						ObjectMeta: metav1.ObjectMeta{
							Finalizers: []string{},
						},
						Spec: nomosv1alpha1.SyncSpec{
							Groups: []nomosv1alpha1.SyncGroup{
								{
									Group: "",
									Kinds: []nomosv1alpha1.SyncKind{
										{
											Kind: "ConfigMap",
											Versions: []nomosv1alpha1.SyncVersion{
												{
													Version: "v1",
												},
											},
										},
									},
								},
								{
									Group: "",
									Kinds: []nomosv1alpha1.SyncKind{
										{
											Kind: "Deployment",
											Versions: []nomosv1alpha1.SyncVersion{
												{
													Version: "v1",
												},
											},
										},
									},
								},
								{
									Group: "rbac.authorization.k8s.io",
									Kinds: []nomosv1alpha1.SyncKind{
										{
											Kind: "Role",
											Versions: []nomosv1alpha1.SyncVersion{
												{
													Version: "v1",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantStatusUpdates: []nomosv1alpha1.Sync{
				{
					ObjectMeta: metav1.ObjectMeta{
						Finalizers: []string{},
					},
					Spec: nomosv1alpha1.SyncSpec{
						Groups: []nomosv1alpha1.SyncGroup{
							{
								Group: "",
								Kinds: []nomosv1alpha1.SyncKind{
									{
										Kind: "ConfigMap",
										Versions: []nomosv1alpha1.SyncVersion{
											{
												Version: "v1",
											},
										},
									},
								},
							},
							{
								Group: "",
								Kinds: []nomosv1alpha1.SyncKind{
									{
										Kind: "Deployment",
										Versions: []nomosv1alpha1.SyncVersion{
											{
												Version: "v1",
											},
										},
									},
								},
							},
							{
								Group: "rbac.authorization.k8s.io",
								Kinds: []nomosv1alpha1.SyncKind{
									{
										Kind: "Role",
										Versions: []nomosv1alpha1.SyncVersion{
											{
												Version: "v1",
											},
										},
									},
								},
							},
						},
					},
					Status: nomosv1alpha1.SyncStatus{
						GroupVersionKinds: []nomosv1alpha1.SyncGroupVersionKindStatus{
							{
								Group:   "",
								Version: "v1",
								Kind:    "ConfigMap",
								Status:  nomosv1alpha1.Syncing,
							},
							{
								Group:   "",
								Version: "v1",
								Kind:    "Deployment",
								Status:  nomosv1alpha1.Syncing,
							},
							{
								Group:   "rbac.authorization.k8s.io",
								Version: "v1",
								Kind:    "Role",
								Status:  nomosv1alpha1.Syncing,
							},
						},
					},
				},
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
			name: "don't update state for multiple Group/Kinds in one sync when unnecessary",
			actualSyncs: nomosv1alpha1.SyncList{
				Items: []nomosv1alpha1.Sync{
					{
						ObjectMeta: metav1.ObjectMeta{
							Finalizers: []string{},
						},
						Spec: nomosv1alpha1.SyncSpec{
							Groups: []nomosv1alpha1.SyncGroup{
								{
									Group: "",
									Kinds: []nomosv1alpha1.SyncKind{
										{
											Kind: "ConfigMap",
											Versions: []nomosv1alpha1.SyncVersion{
												{
													Version: "v1",
												},
											},
										},
									},
								},
								{
									Group: "",
									Kinds: []nomosv1alpha1.SyncKind{
										{
											Kind: "Deployment",
											Versions: []nomosv1alpha1.SyncVersion{
												{
													Version: "v1",
												},
											},
										},
									},
								},
								{
									Group: "rbac.authorization.k8s.io",
									Kinds: []nomosv1alpha1.SyncKind{
										{
											Kind: "Role",
											Versions: []nomosv1alpha1.SyncVersion{
												{
													Version: "v1",
												},
											},
										},
									},
								},
							},
						},
						Status: nomosv1alpha1.SyncStatus{
							GroupVersionKinds: []nomosv1alpha1.SyncGroupVersionKindStatus{
								{
									Group:   "",
									Version: "v1",
									Kind:    "ConfigMap",
									Status:  nomosv1alpha1.Syncing,
								},
								{
									Group:   "",
									Version: "v1",
									Kind:    "Deployment",
									Status:  nomosv1alpha1.Syncing,
								},
								{
									Group:   "rbac.authorization.k8s.io",
									Version: "v1",
									Kind:    "Role",
									Status:  nomosv1alpha1.Syncing,
								},
							},
						},
					},
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
			wantUpdates: []nomosv1alpha1.Sync{
				withDeleteTimestamp(sync("", "v1", "Deployment", nomosv1alpha1.Syncing)),
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
				client:                 client.New(mockClient),
				cache:                  mockCache,
				genericResourceManager: mockManager,
			}

			mockCache.EXPECT().
				List(gomock.Any(), gomock.Any(), gomock.Any()).
				SetArg(2, tc.actualSyncs)

			mockManager.EXPECT().UpdateSyncResources(gomock.Any(), gomock.Any())
			for _, wantUpdate := range tc.wantUpdates {
				// Updates involve first getting the resource from API Server.
				mockClient.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any())
				mockClient.EXPECT().
					Update(gomock.Any(), gomock.Eq(&wantUpdate))
			}

			mockClient.EXPECT().Status().Times(len(tc.wantStatusUpdates)).Return(mockStatusClient)
			for _, wantStatusUpdate := range tc.wantStatusUpdates {
				// Updates involve first getting the resource from API Server.
				mockClient.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any())
				mockStatusClient.EXPECT().
					Update(gomock.Any(), gomock.Eq(&wantStatusUpdate))
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
