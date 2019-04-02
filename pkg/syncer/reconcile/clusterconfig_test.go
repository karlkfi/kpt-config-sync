/*
Copyright 2018 The CSP Config Management Authors.

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
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/syncer/client"
	syncertesting "github.com/google/nomos/pkg/syncer/testing"
	"github.com/google/nomos/pkg/testing/fake"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const token = "b38239ea8f58eaed17af6734bd6a025eeafccda1"

var (
	clusterReconcileComplete = &event{
		kind:    corev1.EventTypeNormal,
		reason:  "ReconcileComplete",
		varargs: true,
		obj:     clusterCfg,
	}
)

func importToken(t string) object.Mutator {
	return func(o *ast.FileObject) {
		switch obj := o.Object.(type) {
		case *v1.ClusterConfig:
			obj.Spec.Token = t
		case *v1.NamespaceConfig:
			obj.Spec.Token = t
		default:
			panic(fmt.Sprintf("Invalid type %T", obj))
		}
	}
}

func syncTime(t metav1.Time) object.Mutator {
	return func(o *ast.FileObject) {
		switch obj := o.Object.(type) {
		case *v1.ClusterConfig:
			obj.Status.SyncTime = t
		case *v1.NamespaceConfig:
			obj.Status.SyncTime = t
		default:
			panic(fmt.Sprintf("Invalid type %T", obj))
		}
	}
}

func syncToken(t string) object.Mutator {
	return func(o *ast.FileObject) {
		switch obj := o.Object.(type) {
		case *v1.ClusterConfig:
			obj.Status.Token = t
		case *v1.NamespaceConfig:
			obj.Status.Token = t
		default:
			panic(fmt.Sprintf("Invalid type %T", obj))
		}
	}
}

func clusterSyncError(err v1.ConfigManagementError) object.Mutator {
	return func(o *ast.FileObject) {
		o.Object.(*v1.ClusterConfig).Status.SyncErrors = append(o.Object.(*v1.ClusterConfig).Status.SyncErrors, err)
	}
}

func clusterConfig(state v1.PolicySyncState, opts ...object.Mutator) *v1.ClusterConfig {
	opts = append(opts, func(o *ast.FileObject) {
		o.Object.(*v1.ClusterConfig).Status.SyncState = state
	})
	return fake.Build(kinds.ClusterConfig(), opts...).Object.(*v1.ClusterConfig)
}

func persistentVolume(reclaimPolicy corev1.PersistentVolumeReclaimPolicy, opts ...object.Mutator) *corev1.PersistentVolume {
	opts = append(opts, func(o *ast.FileObject) {
		o.Object.(*corev1.PersistentVolume).Spec.PersistentVolumeReclaimPolicy = reclaimPolicy
	}, herringAnnotation)
	return fake.Build(kinds.PersistentVolume(), opts...).Object.(*corev1.PersistentVolume)
}

var (
	tokenAnnotation = object.Annotation(v1.SyncTokenAnnotationKey, token)

	// herringAnnotation is used when the decoder mangles empty vs. nil map.
	herringAnnotation = object.Annotation("red", "herring")

	clusterCfg       = clusterConfig(v1.StateSynced, importToken(token))
	clusterCfgSynced = clusterConfig(v1.StateSynced, importToken(token), syncTime(metav1.Time{Time: time.Unix(0, 0)}), syncToken(token))

	managementEnabled  = object.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled)
	managementDisabled = object.Annotation(v1.ResourceManagementKey, v1.ResourceManagementDisabled)
	managementInvalid  = object.Annotation(v1.ResourceManagementKey, "invalid")

	converter = runtime.NewTestUnstructuredConverter(conversion.EqualitiesOrDie())

	anyContext = gomock.Any()
	anyMessage = gomock.Any()
	anyArgs    = gomock.Any()
)

func newTestMocks(t *testing.T, mockCtrl *gomock.Controller) testMocks {
	return testMocks{
		t:            t,
		mockCtrl:     mockCtrl,
		mockClient:   syncertesting.NewMockClient(mockCtrl),
		mockApplier:  syncertesting.NewMockApplier(mockCtrl),
		mockCache:    syncertesting.NewMockGenericCache(mockCtrl),
		mockRecorder: syncertesting.NewMockEventRecorder(mockCtrl),
	}
}

type testMocks struct {
	t            *testing.T
	mockCtrl     *gomock.Controller
	mockClient   *syncertesting.MockClient
	mockApplier  *syncertesting.MockApplier
	mockCache    *syncertesting.MockGenericCache
	mockRecorder *syncertesting.MockEventRecorder
}

func (tm *testMocks) expectClusterCacheGet(config *v1.ClusterConfig) {
	if config == nil {
		return
	}
	tm.mockCache.EXPECT().Get(
		anyContext, types.NamespacedName{Name: config.Name}, EqN(tm.t, "ClusterCacheGet", &v1.ClusterConfig{})).
		SetArg(2, *config)
}

// expectNamespaceCacheGet organizes the mock calls for first retrieving a NamespaceConfig from the
// cache, and then supplying the Namespace. Correctly returns the intermediate not found error if
// supplied nil for a config.
//
// Does not yet support missing Namespace.
func (tm *testMocks) expectNamespaceCacheGet(config *v1.NamespaceConfig, namespace *corev1.Namespace) {
	if config == nil {
		tm.mockCache.EXPECT().Get(
			anyContext, types.NamespacedName{Name: namespace.Name}, EqN(tm.t, "NamespaceConfigCacheGet", &v1.NamespaceConfig{})).
			Return(errors.NewNotFound(schema.GroupResource{}, ""))
	} else {
		tm.mockCache.EXPECT().Get(
			anyContext, types.NamespacedName{Name: namespace.Name}, EqN(tm.t, "NamespaceConfigCacheGet", &v1.NamespaceConfig{})).
			SetArg(2, *config)
	}
	tm.mockCache.EXPECT().Get(
		anyContext, types.NamespacedName{Name: namespace.Name}, EqN(tm.t, "NamespaceCacheGet", &corev1.Namespace{})).
		SetArg(2, *namespace)
}

func (tm *testMocks) expectNamespaceUpdate(namespace *corev1.Namespace) {
	if namespace == nil {
		return
	}
	tm.mockClient.EXPECT().Update(
		anyContext, EqN(tm.t, "NamespaceUpdate", namespace))
}

func (tm *testMocks) expectClusterClientGet(config *v1.ClusterConfig) {
	if config == nil {
		return
	}
	tm.mockClient.EXPECT().Get(
		anyContext, types.NamespacedName{Name: config.Name}, Eq(tm.t, config))
}

func (tm *testMocks) expectNamespaceConfigClientGet(config *v1.NamespaceConfig) {
	if config == nil {
		return
	}
	tm.mockClient.EXPECT().Get(
		anyContext, types.NamespacedName{Name: config.Name}, EqN(tm.t, "NamespaceConfigClientGet", config))
}

func (tm *testMocks) expectNamespaceClientGet(namespace *corev1.Namespace) {
	if namespace == nil {
		return
	}
	tm.mockClient.EXPECT().Get(
		anyContext, types.NamespacedName{Name: namespace.Name}, EqN(tm.t, "NamespaceClientGet", namespace))
}

func (tm *testMocks) expectCacheList(gvk schema.GroupVersionKind, namespace string, obj ...runtime.Object) {
	tm.mockCache.EXPECT().
		UnstructuredList(Eq(tm.t, gvk), Eq(tm.t, namespace)).
		Return(toUnstructuredList(tm.t, converter, obj...), nil)
}

func (tm *testMocks) expectCreate(obj runtime.Object) {
	if obj == nil {
		return
	}
	tm.mockApplier.EXPECT().
		Create(anyContext, Eq(tm.t, toUnstructured(tm.t, converter, obj))).
		Return(true, nil)
}

func (tm *testMocks) expectUpdate(d *diff) {
	if d == nil {
		return
	}
	declared := toUnstructured(tm.t, converter, d.declared)
	actual := toUnstructured(tm.t, converter, d.actual)

	tm.mockApplier.EXPECT().
		Update(anyContext, Eq(tm.t, declared), Eq(tm.t, actual)).
		Return(true, nil)
}

func (tm *testMocks) expectDelete(obj runtime.Object) {
	if obj == nil {
		return
	}
	tm.mockApplier.EXPECT().
		Delete(anyContext, Eq(tm.t, toUnstructured(tm.t, converter, obj))).
		Return(true, nil)
}

func (tm *testMocks) expectEvent(event *event) {
	if event == nil {
		return
	}
	if event.varargs {
		tm.mockRecorder.EXPECT().
			Eventf(Eq(tm.t, event.obj), Eq(tm.t, event.kind), Eq(tm.t, event.reason), anyMessage, anyArgs)
	} else {
		tm.mockRecorder.EXPECT().
			Event(Eq(tm.t, event.obj), Eq(tm.t, event.kind), Eq(tm.t, event.reason), anyMessage)
	}
}

func (tm *testMocks) expectClusterStatusUpdate(statusUpdate *v1.ClusterConfig) {
	if statusUpdate == nil {
		return
	}
	mockStatusClient := syncertesting.NewMockStatusWriter(tm.mockCtrl)
	tm.mockClient.EXPECT().Status().Return(mockStatusClient)
	mockStatusClient.EXPECT().Update(anyContext, Eq(tm.t, statusUpdate))
}

func (tm *testMocks) expectNamespaceStatusUpdate(statusUpdate *v1.NamespaceConfig) {
	if statusUpdate == nil {
		return
	}
	mockStatusClient := syncertesting.NewMockStatusWriter(tm.mockCtrl)
	tm.mockClient.EXPECT().Status().Return(mockStatusClient)
	mockStatusClient.EXPECT().Update(anyContext, Eq(tm.t, statusUpdate))
}

type diff struct {
	declared runtime.Object
	actual   runtime.Object
}

func TestClusterConfigReconcile(t *testing.T) {
	now = func() metav1.Time {
		return metav1.Time{Time: time.Unix(0, 0)}
	}

	testCases := []struct {
		name               string
		actual             runtime.Object
		declared           runtime.Object
		expectCreate       runtime.Object
		expectUpdate       *diff
		expectDelete       runtime.Object
		expectStatusUpdate *v1.ClusterConfig
		expectEvent        *event
	}{
		{
			name:               "create from declared state",
			declared:           persistentVolume(corev1.PersistentVolumeReclaimRecycle),
			expectCreate:       persistentVolume(corev1.PersistentVolumeReclaimRecycle, tokenAnnotation, managementEnabled),
			expectStatusUpdate: clusterCfgSynced,
			expectEvent:        clusterReconcileComplete,
		},
		{
			name:               "do not create if management disabled",
			declared:           persistentVolume(corev1.PersistentVolumeReclaimRecycle, managementDisabled),
			expectStatusUpdate: clusterCfgSynced,
		},
		{
			name:               "do not create if management invalid",
			declared:           persistentVolume(corev1.PersistentVolumeReclaimRecycle, managementInvalid),
			expectStatusUpdate: clusterCfgSynced,
			expectEvent: &event{
				kind:   corev1.EventTypeWarning,
				reason: "InvalidAnnotation",
				obj:    toUnstructured(t, converter, persistentVolume(corev1.PersistentVolumeReclaimRecycle, managementInvalid, tokenAnnotation)),
			},
		},
		{
			name:     "update to declared state",
			declared: persistentVolume(corev1.PersistentVolumeReclaimRecycle),
			actual:   persistentVolume(corev1.PersistentVolumeReclaimDelete, managementEnabled),
			expectUpdate: &diff{
				declared: persistentVolume(corev1.PersistentVolumeReclaimRecycle, tokenAnnotation, managementEnabled),
				actual:   persistentVolume(corev1.PersistentVolumeReclaimDelete, managementEnabled),
			},
			expectStatusUpdate: clusterCfgSynced,
			expectEvent:        clusterReconcileComplete,
		},
		{
			name:     "update to declared state even if actual managed unset",
			declared: persistentVolume(corev1.PersistentVolumeReclaimRecycle),
			actual:   persistentVolume(corev1.PersistentVolumeReclaimDelete),
			expectUpdate: &diff{
				declared: persistentVolume(corev1.PersistentVolumeReclaimRecycle, tokenAnnotation, managementEnabled),
				actual:   persistentVolume(corev1.PersistentVolumeReclaimDelete),
			},
			expectStatusUpdate: clusterCfgSynced,
			expectEvent:        clusterReconcileComplete,
		},
		{
			name:     "update to declared state even if actual managed invalid",
			declared: persistentVolume(corev1.PersistentVolumeReclaimRecycle),
			actual:   persistentVolume(corev1.PersistentVolumeReclaimDelete, managementInvalid),
			expectUpdate: &diff{
				declared: persistentVolume(corev1.PersistentVolumeReclaimRecycle, tokenAnnotation, managementEnabled),
				actual:   persistentVolume(corev1.PersistentVolumeReclaimDelete, managementInvalid),
			},
			expectStatusUpdate: clusterCfgSynced,
			expectEvent:        clusterReconcileComplete,
		},
		{
			name:               "do not update if declared management invalid",
			declared:           persistentVolume(corev1.PersistentVolumeReclaimRecycle, managementInvalid),
			actual:             persistentVolume(corev1.PersistentVolumeReclaimDelete),
			expectStatusUpdate: clusterCfgSynced,
			expectEvent: &event{
				kind:   corev1.EventTypeWarning,
				reason: "InvalidAnnotation",
				obj:    toUnstructured(t, converter, persistentVolume(corev1.PersistentVolumeReclaimRecycle, managementInvalid, tokenAnnotation)),
			},
		},
		{
			name:     "update to unmanaged",
			declared: persistentVolume(corev1.PersistentVolumeReclaimRecycle, managementDisabled),
			actual:   persistentVolume(corev1.PersistentVolumeReclaimDelete, managementEnabled),
			expectUpdate: &diff{
				declared: persistentVolume(corev1.PersistentVolumeReclaimDelete),
				actual:   persistentVolume(corev1.PersistentVolumeReclaimDelete, managementEnabled),
			},
			expectStatusUpdate: clusterCfgSynced,
			expectEvent:        clusterReconcileComplete,
		},
		{
			name:               "do not update if unmanaged",
			declared:           persistentVolume(corev1.PersistentVolumeReclaimRecycle, managementDisabled),
			actual:             persistentVolume(corev1.PersistentVolumeReclaimDelete),
			expectStatusUpdate: clusterCfgSynced,
		},
		{
			name:               "delete if managed",
			actual:             persistentVolume(corev1.PersistentVolumeReclaimDelete, managementEnabled),
			expectDelete:       persistentVolume(corev1.PersistentVolumeReclaimDelete, managementEnabled),
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
			actual: persistentVolume(corev1.PersistentVolumeReclaimDelete, managementInvalid),
			expectUpdate: &diff{
				declared: persistentVolume(corev1.PersistentVolumeReclaimDelete),
				actual:   persistentVolume(corev1.PersistentVolumeReclaimDelete, managementInvalid),
			},
			expectStatusUpdate: clusterCfgSynced,
			expectEvent:        clusterReconcileComplete,
		},
		{
			name: "resource with owner reference is ignored",
			actual: persistentVolume(corev1.PersistentVolumeReclaimRecycle, managementEnabled,
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

				tm := newTestMocks(t, mockCtrl)
				fakeDecoder := syncertesting.NewFakeDecoder(toUnstructuredList(t, converter, tc.declared))
				testReconciler := NewClusterConfigReconciler(ctx,
					client.New(tm.mockClient), tm.mockApplier, tm.mockCache, tm.mockRecorder, fakeDecoder, toSync)

				tm.expectClusterCacheGet(clusterCfg)
				tm.expectCacheList(kinds.PersistentVolume(), "", tc.actual)

				tm.expectCreate(tc.expectCreate)
				tm.expectUpdate(tc.expectUpdate)
				tm.expectDelete(tc.expectDelete)

				tm.expectClusterClientGet(clusterCfg)
				tm.expectClusterStatusUpdate(tc.expectStatusUpdate)
				tm.expectEvent(tc.expectEvent)

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
	now = func() metav1.Time {
		return metav1.Time{Time: time.Unix(0, 0)}
	}
	testCases := []struct {
		name             string
		clusterConfig    *v1.ClusterConfig
		wantStatusUpdate *v1.ClusterConfig
		wantEvent        *event
	}{
		{
			name:          "error on clusterconfig with invalid name",
			clusterConfig: clusterConfig(v1.StateSynced, object.Name("some-incorrect-name")),
			wantStatusUpdate: clusterConfig(v1.StateError,
				object.Name("some-incorrect-name"),
				syncTime(now()),
				clusterSyncError(v1.ConfigManagementError{
					ResourceName: "some-incorrect-name",
					ResourceGVK:  v1.SchemeGroupVersion.WithKind("ClusterConfig"),
					ErrorMessage: `ClusterConfig resource has invalid name "some-incorrect-name". To fix, delete the ClusterConfig.`,
				}),
			),
			wantEvent: &event{
				kind:    corev1.EventTypeWarning,
				reason:  "InvalidClusterConfig",
				varargs: true,
				obj:     clusterConfig(v1.StateSynced, object.Name("some-incorrect-name")),
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

			tm := newTestMocks(t, mockCtrl)
			fakeDecoder := syncertesting.NewFakeDecoder(toUnstructuredList(t, converter, nil))
			testReconciler := NewClusterConfigReconciler(ctx,
				client.New(tm.mockClient), tm.mockApplier, tm.mockCache, tm.mockRecorder, fakeDecoder, toSync)

			tm.expectClusterCacheGet(tc.clusterConfig)

			tm.expectClusterClientGet(tc.clusterConfig)
			tm.expectClusterStatusUpdate(tc.wantStatusUpdate)
			tm.expectEvent(tc.wantEvent)

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
