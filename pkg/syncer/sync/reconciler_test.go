package sync

import (
	"context"
	"testing"
	"time"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/kinds"
	syncerclient "github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/metrics"
	"github.com/google/nomos/pkg/syncer/syncertest"
	"github.com/google/nomos/pkg/syncer/syncertest/fake"
	corev1 "k8s.io/api/core/v1"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile(t *testing.T) {
	tcs := []struct {
		name                 string
		actual               []v1.Sync
		want                 []v1.Sync
		reconcileRequestName string
		wantForceRestart     bool
	}{
		{
			name: "update state for one sync",
			actual: []v1.Sync{
				makeSync(kinds.Deployment().GroupKind(), ""),
			},
			want: []v1.Sync{
				makeSync(kinds.Deployment().GroupKind(), v1.Syncing),
			},
		},
		{
			name: "update state for multiple syncs",
			actual: []v1.Sync{
				makeSync(kinds.Role().GroupKind(), ""),
				makeSync(kinds.Deployment().GroupKind(), ""),
				makeSync(kinds.ConfigMap().GroupKind(), ""),
			},
			want: []v1.Sync{
				makeSync(kinds.Role().GroupKind(), v1.Syncing),
				makeSync(kinds.Deployment().GroupKind(), v1.Syncing),
				makeSync(kinds.ConfigMap().GroupKind(), v1.Syncing),
			},
		},
		{
			name: "don't update state for one sync when unnecessary",
			actual: []v1.Sync{
				makeSync(kinds.Deployment().GroupKind(), v1.Syncing),
			},
			want: []v1.Sync{
				makeSync(kinds.Deployment().GroupKind(), v1.Syncing),
			},
		},
		{
			name: "don't update state for multiple syncs when unnecessary",
			actual: []v1.Sync{
				makeSync(kinds.Role().GroupKind(), v1.Syncing),
				makeSync(kinds.Deployment().GroupKind(), v1.Syncing),
				makeSync(kinds.ConfigMap().GroupKind(), v1.Syncing),
			},
			want: []v1.Sync{
				makeSync(kinds.Role().GroupKind(), v1.Syncing),
				makeSync(kinds.Deployment().GroupKind(), v1.Syncing),
				makeSync(kinds.ConfigMap().GroupKind(), v1.Syncing),
			},
		},
		{
			name: "only update syncs with state change",
			actual: []v1.Sync{
				makeSync(schema.GroupKind{Kind: "Secret"}, v1.Syncing),
				makeSync(schema.GroupKind{Kind: "Service"}, v1.Syncing),
				makeSync(kinds.Deployment().GroupKind(), ""),
			},
			want: []v1.Sync{
				makeSync(schema.GroupKind{Kind: "Secret"}, v1.Syncing),
				makeSync(schema.GroupKind{Kind: "Service"}, v1.Syncing),
				makeSync(kinds.Deployment().GroupKind(), v1.Syncing),
			},
		},
		{
			name: "finalize sync that is pending delete",
			actual: []v1.Sync{
				withDeleteTimestamp(withFinalizer(makeSync(kinds.Deployment().GroupKind(), v1.Syncing))),
			},
			want: []v1.Sync{
				withDeleteTimestamp(makeSync(kinds.Deployment().GroupKind(), v1.Syncing)),
			},
		},
		{
			name:                 "force restart reconcile request restarts SubManager",
			reconcileRequestName: forceRestart,
			actual: []v1.Sync{
				makeSync(kinds.Deployment().GroupKind(), ""),
			},
			want: []v1.Sync{
				makeSync(kinds.Deployment().GroupKind(), v1.Syncing),
			},
			wantForceRestart: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var actual []client.Object
			for i := range tc.actual {
				actual = append(actual, &tc.actual[i])
			}
			fakeClient := fake.NewClient(t, runtime.NewScheme(), actual...)

			discoveryClient := fake.NewDiscoveryClient(
				kinds.ConfigMap(),
				kinds.Deployment(),
				corev1.SchemeGroupVersion.WithKind("Secret"),
				corev1.SchemeGroupVersion.WithKind("Service"),
				kinds.Role(),
				rbacv1beta1.SchemeGroupVersion.WithKind("Role"),
			)
			restartable := &fake.RestartableManagerRecorder{}

			testReconciler := &metaReconciler{
				client:          syncerclient.New(fakeClient, metrics.APICallDuration),
				syncReader:      fakeClient,
				discoveryClient: discoveryClient,
				builder:         newSyncAwareBuilder(),
				subManager:      restartable,
				clientFactory: func() (client.Client, error) {
					return fakeClient, nil
				},
				now: syncertest.Now,
			}

			ctx := context.Background()
			_, err := testReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: apimachinerytypes.NamespacedName{
					Name: tc.reconcileRequestName,
				},
			})

			if err != nil {
				t.Errorf("unexpected reconciliation error: %v", err)
			}

			want := make([]client.Object, len(tc.want))
			for i := range tc.want {
				want[i] = &tc.want[i]
			}
			fakeClient.Check(t, want...)

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
