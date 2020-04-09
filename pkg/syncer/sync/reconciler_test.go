package sync

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/kinds"
	syncerclient "github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/metrics"
	syncertesting "github.com/google/nomos/pkg/syncer/testing"
	"github.com/google/nomos/pkg/syncer/testing/fake"
	"github.com/google/nomos/pkg/syncer/testing/mocks"
	corev1 "k8s.io/api/core/v1"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
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
					makeSync(kinds.Deployment().GroupKind(), ""),
				},
			},
			wantStatusUpdates: []v1.Sync{
				makeSync(kinds.Deployment().GroupKind(), v1.Syncing),
			},
		},
		{
			name: "update state for multiple syncs",
			actualSyncs: v1.SyncList{
				Items: []v1.Sync{
					makeSync(kinds.Role().GroupKind(), ""),
					makeSync(kinds.Deployment().GroupKind(), ""),
					makeSync(kinds.ConfigMap().GroupKind(), ""),
				},
			},
			wantStatusUpdates: []v1.Sync{
				makeSync(kinds.Role().GroupKind(), v1.Syncing),
				makeSync(kinds.Deployment().GroupKind(), v1.Syncing),
				makeSync(kinds.ConfigMap().GroupKind(), v1.Syncing),
			},
		},
		{
			name: "don't update state for one sync when unnecessary",
			actualSyncs: v1.SyncList{
				Items: []v1.Sync{
					makeSync(kinds.Deployment().GroupKind(), v1.Syncing),
				},
			},
		},
		{
			name: "don't update state for multiple syncs when unnecessary",
			actualSyncs: v1.SyncList{
				Items: []v1.Sync{
					makeSync(kinds.Role().GroupKind(), v1.Syncing),
					makeSync(kinds.Deployment().GroupKind(), v1.Syncing),
					makeSync(kinds.ConfigMap().GroupKind(), v1.Syncing),
				},
			},
		},
		{
			name: "only update syncs with state change",
			actualSyncs: v1.SyncList{
				Items: []v1.Sync{
					makeSync(schema.GroupKind{Kind: "Secret"}, v1.Syncing),
					makeSync(schema.GroupKind{Kind: "Service"}, v1.Syncing),
					makeSync(kinds.Deployment().GroupKind(), ""),
				},
			},
			wantStatusUpdates: []v1.Sync{
				makeSync(kinds.Deployment().GroupKind(), v1.Syncing),
			},
		},
		{
			name: "finalize sync that is pending delete",
			actualSyncs: v1.SyncList{
				Items: []v1.Sync{
					withDeleteTimestamp(withFinalizer(makeSync(kinds.Deployment().GroupKind(), v1.Syncing))),
				},
			},
			wantUpdateList: []updateList{
				{
					update: withDeleteTimestamp(makeSync(kinds.Deployment().GroupKind(), v1.Syncing)),
					list:   unstructuredList(kinds.Deployment().GroupVersion().WithKind("DeploymentList")),
				},
			},
		},
		{
			name:                 "force restart reconcile request restarts SubManager",
			reconcileRequestName: ForceRestart,
			actualSyncs: v1.SyncList{
				Items: []v1.Sync{
					makeSync(kinds.Deployment().GroupKind(), ""),
				},
			},
			wantStatusUpdates: []v1.Sync{
				makeSync(kinds.Deployment().GroupKind(), v1.Syncing),
			},
			wantForceRestart: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockClient := mocks.NewMockClient(mockCtrl)

			syncsReader := fake.SyncCacheReader(tc.actualSyncs)
			discoveryClient := fake.NewDiscoveryClient(
				kinds.ConfigMap(),
				kinds.Deployment(),
				corev1.SchemeGroupVersion.WithKind("Secret"),
				corev1.SchemeGroupVersion.WithKind("Service"),
				kinds.Role(),
				rbacv1beta1.SchemeGroupVersion.WithKind("Role"),
			)
			restartable := &fake.RestartableManagerRecorder{}

			testReconciler := &MetaReconciler{
				client:          syncerclient.New(mockClient, metrics.APICallDuration),
				syncCache:       syncsReader,
				discoveryClient: discoveryClient,
				builder:         newSyncAwareBuilder(),
				subManager:      restartable,
				clientFactory: func() (client.Client, error) {
					return mockClient, nil
				},
				now: syncertesting.Now,
			}

			for _, wantUpdateList := range tc.wantUpdateList {
				// Updates involve first getting the resource from API Server.
				mockClient.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any())
				mockClient.EXPECT().
					Update(gomock.Any(), gomock.Eq(&wantUpdateList.update))
			}

			statusWriter := fake.StatusWriterRecorder{}
			mockClient.EXPECT().Status().Times(len(tc.wantStatusUpdates)).Return(&statusWriter)
			mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(len(tc.wantStatusUpdates))

			_, err := testReconciler.Reconcile(reconcile.Request{
				NamespacedName: apimachinerytypes.NamespacedName{
					Name: tc.reconcileRequestName,
				},
			})
			if err != nil {
				t.Errorf("unexpected reconciliation error: %v", err)
			}

			wantStatusUpdates := make([]runtime.Object, len(tc.wantStatusUpdates))
			for i, o := range tc.wantStatusUpdates {
				wantStatusUpdates[i] = o.DeepCopyObject()
			}
			statusWriter.Check(t, wantStatusUpdates...)

			if len(restartable.Restarts) != 1 || restartable.Restarts[0] != tc.wantForceRestart {
				t.Errorf("got manager.Restarts = %v, want [%t]", restartable.Restarts, tc.wantForceRestart)
			}
		})
	}
}

func makeSync(gk schema.GroupKind, state v1.SyncState) v1.Sync {
	s := *v1.NewSync(gk)
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
