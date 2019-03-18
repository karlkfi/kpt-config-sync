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

package reconcile

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/syncer/client"
	syncertesting "github.com/google/nomos/pkg/syncer/testing"
	"github.com/google/nomos/pkg/testing/object"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const token = "b38239ea8f58eaed17af6734bd6a025eeafccda1"

var (
	reconcileComplete = event{
		kind:    corev1.EventTypeNormal,
		reason:  "ReconcileComplete",
		varargs: true,
	}
)

func importToken(t string) object.BuildOpt {
	return func(o *ast.FileObject) {
		o.Object.(*v1.ClusterConfig).Spec.ImportToken = t
	}
}

func syncTime(t metav1.Time) object.BuildOpt {
	return func(o *ast.FileObject) {
		o.Object.(*v1.ClusterConfig).Status.SyncTime = t
	}
}

func syncToken(t string) object.BuildOpt {
	return func(o *ast.FileObject) {
		o.Object.(*v1.ClusterConfig).Status.SyncToken = t
	}
}

func syncError(err v1.ClusterConfigSyncError) object.BuildOpt {
	return func(o *ast.FileObject) {
		o.Object.(*v1.ClusterConfig).Status.SyncErrors = append(o.Object.(*v1.ClusterConfig).Status.SyncErrors, err)
	}
}

func clusterConfig(state v1.PolicySyncState, opts ...object.BuildOpt) *v1.ClusterConfig {
	opts = append(opts, func(o *ast.FileObject) {
		o.Object.(*v1.ClusterConfig).Status.SyncState = state
	})
	return object.Build(kinds.ClusterConfig(), opts...).Object.(*v1.ClusterConfig)
}

func persistentVolume(reclaimPolicy corev1.PersistentVolumeReclaimPolicy, opts ...object.BuildOpt) *corev1.PersistentVolume {
	opts = append(opts, func(o *ast.FileObject) {
		o.Object.(*corev1.PersistentVolume).Spec.PersistentVolumeReclaimPolicy = reclaimPolicy
	})
	return object.Build(kinds.PersistentVolume(), opts...).Object.(*corev1.PersistentVolume)
}

func managedPersistentVolume(reclaimPolicy corev1.PersistentVolumeReclaimPolicy, opts ...object.BuildOpt) *corev1.PersistentVolume {
	opts = append(opts,
		object.Annotation(v1.ResourceManagementKey, v1.ResourceManagementValue),
		object.Annotation(v1.SyncTokenAnnotationKey, token))
	return persistentVolume(reclaimPolicy, opts...)
}

func TestClusterConfigReconcile(t *testing.T) {
	now = func() metav1.Time {
		return metav1.Time{Time: time.Unix(0, 0)}
	}
	testCases := []struct {
		name             string
		clusterConfig    *v1.ClusterConfig
		declared         runtime.Object
		actual           runtime.Object
		wantApplies      []application
		wantCreate       runtime.Object
		wantDelete       runtime.Object
		wantStatusUpdate *v1.ClusterConfig
		wantEvents       []event
	}{
		{
			name:          "update actual resource to declared state",
			clusterConfig: clusterConfig(v1.StateSynced, importToken(token)),
			declared:      persistentVolume(corev1.PersistentVolumeReclaimRecycle),
			actual:        managedPersistentVolume(corev1.PersistentVolumeReclaimDelete),
			wantApplies: []application{
				{
					intended: managedPersistentVolume(corev1.PersistentVolumeReclaimRecycle),
					current:  managedPersistentVolume(corev1.PersistentVolumeReclaimDelete),
				},
			},
			wantStatusUpdate: clusterConfig(v1.StateSynced, importToken(token), syncTime(now()), syncToken(token)),
			wantEvents:       []event{reconcileComplete},
		},
		{
			name:          "actual resource already matches declared state",
			clusterConfig: clusterConfig(v1.StateSynced, importToken(token)),
			declared:      persistentVolume(corev1.PersistentVolumeReclaimRecycle),
			actual:        managedPersistentVolume(corev1.PersistentVolumeReclaimRecycle),
			wantApplies: []application{
				{
					intended: managedPersistentVolume(corev1.PersistentVolumeReclaimRecycle),
					current:  managedPersistentVolume(corev1.PersistentVolumeReclaimRecycle),
				},
			},
			wantStatusUpdate: clusterConfig(v1.StateSynced, importToken(token), syncTime(now()), syncToken(token)),
			wantEvents:       []event{reconcileComplete},
		},
		{
			name:          "un-managed resource cannot be synced",
			clusterConfig: clusterConfig(v1.StateSynced),
			declared:      persistentVolume(corev1.PersistentVolumeReclaimRecycle),
			actual:        persistentVolume(corev1.PersistentVolumeReclaimDelete),
		},
		{
			name:             "create resource from declared state",
			clusterConfig:    clusterConfig(v1.StateSynced, importToken(token)),
			declared:         persistentVolume(corev1.PersistentVolumeReclaimRecycle),
			wantCreate:       managedPersistentVolume(corev1.PersistentVolumeReclaimRecycle),
			wantStatusUpdate: clusterConfig(v1.StateSynced, importToken(token), syncTime(now()), syncToken(token)),
			wantEvents:       []event{reconcileComplete},
		},
		{
			name:          "delete resource according to declared state",
			clusterConfig: clusterConfig(v1.StateSynced),
			actual:        managedPersistentVolume(corev1.PersistentVolumeReclaimRecycle),
			wantDelete:    managedPersistentVolume(corev1.PersistentVolumeReclaimRecycle),
			wantEvents:    []event{reconcileComplete},
		},
		{
			name:          "error on clusterconfig with invalid name",
			clusterConfig: clusterConfig(v1.StateSynced, object.Name("some-incorrect-name")),
			wantStatusUpdate: clusterConfig(v1.StateError,
				object.Name("some-incorrect-name"),
				syncTime(now()),
				syncError(v1.ClusterConfigSyncError{
					ResourceName: "some-incorrect-name",
					ResourceKind: "ClusterConfig",
					ResourceAPI:  "configmanagement.gke.io/v1",
					ErrorMessage: `ClusterConfig resource has invalid name "some-incorrect-name"`,
				}),
			),
			wantEvents: []event{
				{
					kind:    corev1.EventTypeWarning,
					reason:  "InvalidClusterConfig",
					varargs: true,
				},
			},
		},
	}

	converter := runtime.NewTestUnstructuredConverter(conversion.EqualitiesOrDie())
	toSync := []schema.GroupVersionKind{kinds.PersistentVolume()}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockClient := syncertesting.NewMockClient(mockCtrl)
			mockApplier := syncertesting.NewMockApplier(mockCtrl)
			mockCache := syncertesting.NewMockGenericCache(mockCtrl)
			mockRecorder := syncertesting.NewMockEventRecorder(mockCtrl)
			var declared []runtime.Object
			if tc.declared != nil {
				declared = append(declared, tc.declared)
			}
			fakeDecoder := syncertesting.NewFakeDecoder(toUnstructureds(t, converter, declared))

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			testReconciler := NewClusterConfigReconciler(ctx,
				client.New(mockClient), mockApplier, mockCache, mockRecorder, fakeDecoder, toSync)

			// Get ClusterConfig from cache.
			mockCache.EXPECT().
				Get(gomock.Any(), gomock.Any(), gomock.Any()).
				SetArg(2, *tc.clusterConfig)

			// List actual resources on the cluster.
			if tc.clusterConfig.Name == v1.ClusterConfigName {
				// No call is made if the cluster config's name is incorrect.
				var expectedActual []runtime.Object
				if tc.actual != nil {
					expectedActual = append(expectedActual, tc.actual)
				}
				mockCache.EXPECT().
					UnstructuredList(gomock.Any(), gomock.Eq("")).
					Return(toUnstructureds(t, converter, expectedActual), nil)
			}

			// Check for expected creates, applies and deletes.
			if tc.wantCreate != nil {
				mockApplier.EXPECT().
					Create(gomock.Any(), NewUnstructuredMatcher(toUnstructured(t, converter, tc.wantCreate)))
			}
			for _, wantApply := range tc.wantApplies {
				mockApplier.EXPECT().
					ApplyCluster(
						gomock.Eq(toUnstructured(t, converter, wantApply.intended)),
						gomock.Eq(toUnstructured(t, converter, wantApply.current)))
			}
			if tc.wantDelete != nil {
				mockClient.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Eq(toUnstructured(t, converter, tc.wantDelete)))
				mockClient.EXPECT().
					Delete(gomock.Any(), gomock.Eq(toUnstructured(t, converter, tc.wantDelete)))
			}

			if tc.wantStatusUpdate != nil {
				// Updates involve first getting the resource from API Server.
				mockClient.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any())
				mockStatusClient := syncertesting.NewMockStatusWriter(mockCtrl)
				mockClient.EXPECT().Status().Return(mockStatusClient)
				mockStatusClient.EXPECT().
					Update(gomock.Any(), gomock.Eq(tc.wantStatusUpdate))
			}

			// Check for events with warning or status.
			for _, wantEvent := range tc.wantEvents {
				mockRecorder.EXPECT().
					Eventf(gomock.Any(), gomock.Eq(wantEvent.kind), gomock.Eq(wantEvent.reason), gomock.Any(), gomock.Any())
			}

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
