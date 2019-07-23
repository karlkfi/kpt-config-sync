package reconcile

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/nomos/pkg/api/configmanagement"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/metrics"
	syncertesting "github.com/google/nomos/pkg/syncer/testing"
	"github.com/google/nomos/pkg/syncer/testing/mocks"
	"github.com/google/nomos/pkg/testing/fake"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	clusterReconcileComplete = &syncertesting.Event{
		Kind:    corev1.EventTypeNormal,
		Reason:  "ReconcileComplete",
		Varargs: true,
		Obj:     clusterCfg,
	}
)

func clusterSyncError(err v1.ConfigManagementError) fake.ClusterConfigMutator {
	return func(cc *v1.ClusterConfig) {
		cc.Status.SyncErrors = append(cc.Status.SyncErrors, err)
	}
}

func clusterConfig(state v1.ConfigSyncState, opts ...fake.ClusterConfigMutator) *v1.ClusterConfig {
	mutators := append(opts, fake.ClusterConfigMeta(syncertesting.Herrings...))
	result := fake.ClusterConfigObject(mutators...)
	result.Status.SyncState = state
	return result
}

func persistentVolume(reclaimPolicy corev1.PersistentVolumeReclaimPolicy, opts ...object.MetaMutator) *corev1.PersistentVolume {
	mutators := append(opts, syncertesting.Herrings...)
	result := fake.PersistentVolumeObject(mutators...)
	result.Spec.PersistentVolumeReclaimPolicy = reclaimPolicy
	return result
}

var (
	clusterCfg       = clusterConfig(v1.StateSynced, syncertesting.ClusterConfigImportToken(syncertesting.Token))
	clusterCfgSynced = clusterConfig(v1.StateSynced, syncertesting.ClusterConfigImportToken(syncertesting.Token),
		syncertesting.ClusterConfigSyncTime(), syncertesting.ClusterConfigSyncToken())
)

func TestClusterConfigReconcile(t *testing.T) {
	testCases := []struct {
		name               string
		actual             runtime.Object
		declared           runtime.Object
		expectCreate       runtime.Object
		expectUpdate       *syncertesting.Diff
		expectDelete       runtime.Object
		expectStatusUpdate *v1.ClusterConfig
		expectEvent        *syncertesting.Event
	}{
		{
			name:     "create from declared state",
			declared: persistentVolume(corev1.PersistentVolumeReclaimRecycle),
			expectCreate: persistentVolume(corev1.PersistentVolumeReclaimRecycle, syncertesting.TokenAnnotation,
				syncertesting.ManagementEnabled),
			expectStatusUpdate: clusterCfgSynced,
			expectEvent:        clusterReconcileComplete,
		},
		{
			name:               "do not create if management disabled",
			declared:           persistentVolume(corev1.PersistentVolumeReclaimRecycle, syncertesting.ManagementDisabled),
			expectStatusUpdate: clusterCfgSynced,
		},
		{
			name:               "do not create if management invalid",
			declared:           persistentVolume(corev1.PersistentVolumeReclaimRecycle, syncertesting.ManagementInvalid),
			expectStatusUpdate: clusterCfgSynced,
			expectEvent: &syncertesting.Event{
				Kind:   corev1.EventTypeWarning,
				Reason: "InvalidAnnotation",
				Obj: syncertesting.ToUnstructured(t, syncertesting.Converter, persistentVolume(corev1.PersistentVolumeReclaimRecycle,
					syncertesting.ManagementInvalid, syncertesting.TokenAnnotation)),
			},
		},
		{
			name:     "update to declared state",
			declared: persistentVolume(corev1.PersistentVolumeReclaimRecycle),
			actual:   persistentVolume(corev1.PersistentVolumeReclaimDelete, syncertesting.ManagementEnabled),
			expectUpdate: &syncertesting.Diff{
				Declared: persistentVolume(corev1.PersistentVolumeReclaimRecycle, syncertesting.TokenAnnotation, syncertesting.ManagementEnabled),
				Actual:   persistentVolume(corev1.PersistentVolumeReclaimDelete, syncertesting.ManagementEnabled),
			},
			expectStatusUpdate: clusterCfgSynced,
			expectEvent:        clusterReconcileComplete,
		},
		{
			name:     "update to declared state even if actual managed unset",
			declared: persistentVolume(corev1.PersistentVolumeReclaimRecycle),
			actual:   persistentVolume(corev1.PersistentVolumeReclaimDelete),
			expectUpdate: &syncertesting.Diff{
				Declared: persistentVolume(corev1.PersistentVolumeReclaimRecycle, syncertesting.TokenAnnotation, syncertesting.ManagementEnabled),
				Actual:   persistentVolume(corev1.PersistentVolumeReclaimDelete),
			},
			expectStatusUpdate: clusterCfgSynced,
			expectEvent:        clusterReconcileComplete,
		},
		{
			name:     "update to declared state even if actual managed invalid",
			declared: persistentVolume(corev1.PersistentVolumeReclaimRecycle),
			actual:   persistentVolume(corev1.PersistentVolumeReclaimDelete, syncertesting.ManagementInvalid),
			expectUpdate: &syncertesting.Diff{
				Declared: persistentVolume(corev1.PersistentVolumeReclaimRecycle, syncertesting.TokenAnnotation, syncertesting.ManagementEnabled),
				Actual:   persistentVolume(corev1.PersistentVolumeReclaimDelete, syncertesting.ManagementInvalid),
			},
			expectStatusUpdate: clusterCfgSynced,
			expectEvent:        clusterReconcileComplete,
		},
		{
			name:               "do not update if declared management invalid",
			declared:           persistentVolume(corev1.PersistentVolumeReclaimRecycle, syncertesting.ManagementInvalid),
			actual:             persistentVolume(corev1.PersistentVolumeReclaimDelete),
			expectStatusUpdate: clusterCfgSynced,
			expectEvent: &syncertesting.Event{
				Kind:   corev1.EventTypeWarning,
				Reason: "InvalidAnnotation",
				Obj: syncertesting.ToUnstructured(t, syncertesting.Converter, persistentVolume(corev1.PersistentVolumeReclaimRecycle,
					syncertesting.ManagementInvalid, syncertesting.TokenAnnotation)),
			},
		},
		{
			name:     "update to unmanaged",
			declared: persistentVolume(corev1.PersistentVolumeReclaimRecycle, syncertesting.ManagementDisabled),
			actual:   persistentVolume(corev1.PersistentVolumeReclaimDelete, syncertesting.ManagementEnabled),
			expectUpdate: &syncertesting.Diff{
				Declared: persistentVolume(corev1.PersistentVolumeReclaimDelete),
				Actual:   persistentVolume(corev1.PersistentVolumeReclaimDelete, syncertesting.ManagementEnabled),
			},
			expectStatusUpdate: clusterCfgSynced,
			expectEvent:        clusterReconcileComplete,
		},
		{
			name:               "do not update if unmanaged",
			declared:           persistentVolume(corev1.PersistentVolumeReclaimRecycle, syncertesting.ManagementDisabled),
			actual:             persistentVolume(corev1.PersistentVolumeReclaimDelete),
			expectStatusUpdate: clusterCfgSynced,
		},
		{
			name:               "delete if managed",
			actual:             persistentVolume(corev1.PersistentVolumeReclaimDelete, syncertesting.ManagementEnabled),
			expectDelete:       persistentVolume(corev1.PersistentVolumeReclaimDelete, syncertesting.ManagementEnabled),
			expectStatusUpdate: clusterCfgSynced,
			expectEvent:        clusterReconcileComplete,
		},
		{
			name:               "do not delete if unmanaged",
			actual:             persistentVolume(corev1.PersistentVolumeReclaimDelete),
			expectStatusUpdate: clusterCfgSynced,
		},
		{
			name:   "unmanage if invalid",
			actual: persistentVolume(corev1.PersistentVolumeReclaimDelete, syncertesting.ManagementInvalid),
			expectUpdate: &syncertesting.Diff{
				Declared: persistentVolume(corev1.PersistentVolumeReclaimDelete),
				Actual:   persistentVolume(corev1.PersistentVolumeReclaimDelete, syncertesting.ManagementInvalid),
			},
			expectStatusUpdate: clusterCfgSynced,
			expectEvent:        clusterReconcileComplete,
		},
		{
			name: "resource with owner reference is ignored",
			actual: persistentVolume(corev1.PersistentVolumeReclaimRecycle, syncertesting.ManagementEnabled,
				object.OwnerReference(
					"some_operator_config_object",
					"some_uid",
					schema.GroupVersionKind{Group: "operator.config.group", Kind: "OperatorConfigObject", Version: "v1"}),
			),
			expectStatusUpdate: clusterCfgSynced,
		},
	}

	toSync := []schema.GroupVersionKind{kinds.PersistentVolume()}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run(tc.name, func(t *testing.T) {
				mockCtrl := gomock.NewController(t)
				defer mockCtrl.Finish()

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				tm := syncertesting.NewTestMocks(t, mockCtrl)
				fakeDecoder := mocks.NewFakeDecoder(syncertesting.ToUnstructuredList(t, syncertesting.Converter, tc.declared))
				testReconciler := NewClusterConfigReconciler(ctx,
					client.New(tm.MockClient, metrics.APICallDuration), tm.MockApplier, tm.MockCache, tm.MockRecorder, fakeDecoder, syncertesting.Now, toSync)

				tm.ExpectClusterCacheGet(clusterCfg)
				tm.ExpectCacheList(kinds.PersistentVolume(), "", tc.actual)

				tm.ExpectCreate(tc.expectCreate)
				tm.ExpectUpdate(tc.expectUpdate)
				tm.ExpectDelete(tc.expectDelete)

				tm.ExpectClusterClientGet(clusterCfg)
				tm.ExpectClusterStatusUpdate(tc.expectStatusUpdate)
				tm.ExpectEvent(tc.expectEvent)

				_, err := testReconciler.Reconcile(
					reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name: v1.ClusterConfigName,
						},
					})
				if err != nil {
					t.Errorf("unexpected reconciliation error: %v", err)
				}
			})
		})
	}
}

func TestInvalidClusterConfig(t *testing.T) {
	testCases := []struct {
		name             string
		clusterConfig    *v1.ClusterConfig
		wantStatusUpdate *v1.ClusterConfig
		wantEvent        *syncertesting.Event
	}{
		{
			name:          "error on clusterconfig with invalid name",
			clusterConfig: clusterConfig(v1.StateSynced, fake.ClusterConfigMeta(object.Name("some-incorrect-name"))),
			wantStatusUpdate: clusterConfig(v1.StateError,
				fake.ClusterConfigMeta(object.Name("some-incorrect-name")),
				syncertesting.ClusterConfigSyncTime(),
				clusterSyncError(v1.ConfigManagementError{
					ErrorResources: []v1.ErrorResource{
						{
							ResourceName: "some-incorrect-name",
							ResourceGVK:  v1.SchemeGroupVersion.WithKind(configmanagement.ClusterConfigKind),
						},
					},
					ErrorMessage: `ClusterConfig resource has invalid name "some-incorrect-name". To fix, delete the ClusterConfig.`,
				}),
			),
			wantEvent: &syncertesting.Event{
				Kind:    corev1.EventTypeWarning,
				Reason:  "InvalidClusterConfig",
				Varargs: true,
				Obj:     clusterConfig(v1.StateSynced, fake.ClusterConfigMeta(object.Name("some-incorrect-name"))),
			},
		},
	}

	toSync := []schema.GroupVersionKind{kinds.PersistentVolume()}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			tm := syncertesting.NewTestMocks(t, mockCtrl)
			fakeDecoder := mocks.NewFakeDecoder(syncertesting.ToUnstructuredList(t, syncertesting.Converter, nil))
			testReconciler := NewClusterConfigReconciler(ctx,
				client.New(tm.MockClient, metrics.APICallDuration), tm.MockApplier, tm.MockCache, tm.MockRecorder, fakeDecoder, syncertesting.Now, toSync)

			tm.ExpectClusterCacheGet(tc.clusterConfig)

			tm.ExpectClusterClientGet(tc.clusterConfig)
			tm.ExpectClusterStatusUpdate(tc.wantStatusUpdate)
			tm.ExpectEvent(tc.wantEvent)

			_, err := testReconciler.Reconcile(
				reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name: tc.clusterConfig.Name,
					},
				})
			if err != nil {
				t.Errorf("unexpected reconciliation error: %v", err)
			}
		})
	}
}
