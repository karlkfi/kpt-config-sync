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
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/syncer/client"
	syncertesting "github.com/google/nomos/pkg/syncer/testing"
	"github.com/google/nomos/pkg/syncer/testing/mocks"
	"github.com/google/nomos/pkg/testing/fake"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func deployment(deploymentStrategy appsv1.DeploymentStrategyType, opts ...object.Mutator) *appsv1.Deployment {
	opts = append(opts, func(o *ast.FileObject) {
		o.Object.(*appsv1.Deployment).Spec.Strategy.Type = deploymentStrategy
	}, syncertesting.Herrings)
	return fake.Build(kinds.Deployment(), opts...).Object.(*appsv1.Deployment)
}

func namespaceConfig(name string, state v1.PolicySyncState, opts ...object.Mutator) *v1.NamespaceConfig {
	opts = append(opts, object.Name(name), func(o *ast.FileObject) {
		o.Object.(*v1.NamespaceConfig).Status.SyncState = state
	})
	return fake.Build(kinds.NamespaceConfig(), opts...).Object.(*v1.NamespaceConfig)
}

func namespace(name string, opts ...object.Mutator) *corev1.Namespace {
	opts = append(opts, object.Name(name))
	return fake.Build(kinds.Namespace(), opts...).Object.(*corev1.Namespace)
}

func namespaceSyncError(err v1.ConfigManagementError) object.Mutator {
	return func(o *ast.FileObject) {
		o.Object.(*v1.NamespaceConfig).Status.SyncErrors = append(o.Object.(*v1.NamespaceConfig).Status.SyncErrors, err)
	}
}

var managedQuotaLabels = object.Label(v1.ConfigManagementQuotaKey, v1.ConfigManagementQuotaValue)

var (
	eng                = "eng"
	managedNamespace   = namespace(eng, syncertesting.ManagementEnabled, managedQuotaLabels, syncertesting.TokenAnnotation)
	unmanagedNamespace = namespace(eng)

	namespaceCfg       = namespaceConfig(eng, v1.StateSynced, syncertesting.ImportToken(syncertesting.Token))
	namespaceCfgSynced = namespaceConfig(eng, v1.StateSynced, syncertesting.ImportToken(syncertesting.Token),
		syncertesting.SyncTime(), syncertesting.SyncToken())

	managedNamespaceReconcileComplete = &syncertesting.Event{
		Kind:    corev1.EventTypeNormal,
		Reason:  "ReconcileComplete",
		Varargs: true,
		Obj:     namespaceCfg,
	}
)

func TestManagedNamespaceConfigReconcile(t *testing.T) {
	testCases := []struct {
		name               string
		declared           runtime.Object
		actual             runtime.Object
		expectCreate       runtime.Object
		expectUpdate       *syncertesting.Diff
		expectDelete       runtime.Object
		expectStatusUpdate *v1.NamespaceConfig
		expectEvent        *syncertesting.Event
	}{
		{
			name:               "create from declared state",
			declared:           deployment(appsv1.RollingUpdateDeploymentStrategyType),
			expectCreate:       deployment(appsv1.RollingUpdateDeploymentStrategyType, object.Namespace(eng), syncertesting.ManagementEnabled, syncertesting.TokenAnnotation),
			expectStatusUpdate: namespaceCfgSynced,
			expectEvent:        managedNamespaceReconcileComplete,
		},
		{
			name:               "do not create if management disabled",
			declared:           deployment(appsv1.RollingUpdateDeploymentStrategyType, syncertesting.ManagementDisabled),
			expectStatusUpdate: namespaceCfgSynced,
		},
		{
			name:               "do not create if management invalid",
			declared:           deployment(appsv1.RollingUpdateDeploymentStrategyType, syncertesting.ManagementInvalid),
			expectStatusUpdate: namespaceCfgSynced,
			expectEvent: &syncertesting.Event{
				Kind:   corev1.EventTypeWarning,
				Reason: "InvalidAnnotation",
				Obj: syncertesting.ToUnstructured(t, syncertesting.Converter, deployment(appsv1.RollingUpdateDeploymentStrategyType,
					syncertesting.ManagementInvalid, syncertesting.TokenAnnotation, object.Namespace(eng))),
			},
		},
		{
			name:     "update to declared state",
			declared: deployment(appsv1.RollingUpdateDeploymentStrategyType),
			actual:   deployment(appsv1.RecreateDeploymentStrategyType, syncertesting.ManagementEnabled),
			expectUpdate: &syncertesting.Diff{
				Declared: deployment(appsv1.RollingUpdateDeploymentStrategyType, object.Namespace(eng), syncertesting.ManagementEnabled, syncertesting.TokenAnnotation),
				Actual:   deployment(appsv1.RecreateDeploymentStrategyType, syncertesting.ManagementEnabled),
			},
			expectStatusUpdate: namespaceCfgSynced,
			expectEvent:        managedNamespaceReconcileComplete,
		},
		{
			name:     "update to declared state even if actual managed unset",
			declared: deployment(appsv1.RollingUpdateDeploymentStrategyType),
			actual:   deployment(appsv1.RecreateDeploymentStrategyType),
			expectUpdate: &syncertesting.Diff{
				Declared: deployment(appsv1.RollingUpdateDeploymentStrategyType, object.Namespace(eng), syncertesting.ManagementEnabled, syncertesting.TokenAnnotation),
				Actual:   deployment(appsv1.RecreateDeploymentStrategyType),
			},
			expectStatusUpdate: namespaceCfgSynced,
			expectEvent:        managedNamespaceReconcileComplete,
		},
		{
			name:     "update to declared state even if actual managed invalid",
			declared: deployment(appsv1.RollingUpdateDeploymentStrategyType),
			actual:   deployment(appsv1.RecreateDeploymentStrategyType, syncertesting.ManagementInvalid),
			expectUpdate: &syncertesting.Diff{
				Declared: deployment(appsv1.RollingUpdateDeploymentStrategyType, object.Namespace(eng), syncertesting.ManagementEnabled, syncertesting.TokenAnnotation),
				Actual:   deployment(appsv1.RecreateDeploymentStrategyType, syncertesting.ManagementInvalid),
			},
			expectStatusUpdate: namespaceCfgSynced,
			expectEvent:        managedNamespaceReconcileComplete,
		},
		{
			name:               "do not update if declared managed invalid",
			declared:           deployment(appsv1.RollingUpdateDeploymentStrategyType, syncertesting.ManagementInvalid),
			actual:             deployment(appsv1.RecreateDeploymentStrategyType),
			expectStatusUpdate: namespaceCfgSynced,
			expectEvent: &syncertesting.Event{
				Kind:   corev1.EventTypeWarning,
				Reason: "InvalidAnnotation",
				Obj: syncertesting.ToUnstructured(t, syncertesting.Converter, deployment(appsv1.RollingUpdateDeploymentStrategyType,
					syncertesting.ManagementInvalid, syncertesting.TokenAnnotation, object.Namespace(eng))),
			},
		},
		{
			name:     "update to unmanaged",
			declared: deployment(appsv1.RollingUpdateDeploymentStrategyType, syncertesting.ManagementDisabled),
			actual:   deployment(appsv1.RecreateDeploymentStrategyType, syncertesting.ManagementEnabled),
			expectUpdate: &syncertesting.Diff{
				Declared: deployment(appsv1.RecreateDeploymentStrategyType),
				Actual:   deployment(appsv1.RecreateDeploymentStrategyType, syncertesting.ManagementEnabled),
			},
			expectStatusUpdate: namespaceCfgSynced,
			expectEvent:        managedNamespaceReconcileComplete,
		},
		{
			name:               "do not update if unmanaged",
			declared:           deployment(appsv1.RollingUpdateDeploymentStrategyType, syncertesting.ManagementDisabled),
			actual:             deployment(appsv1.RecreateDeploymentStrategyType),
			expectStatusUpdate: namespaceCfgSynced,
		},
		{
			name:               "delete if managed",
			actual:             deployment(appsv1.RecreateDeploymentStrategyType, syncertesting.ManagementEnabled),
			expectDelete:       deployment(appsv1.RecreateDeploymentStrategyType, syncertesting.ManagementEnabled),
			expectStatusUpdate: namespaceCfgSynced,
			expectEvent:        managedNamespaceReconcileComplete,
		},
		{
			name:               "do not delete if unmanaged",
			actual:             deployment(appsv1.RecreateDeploymentStrategyType),
			expectStatusUpdate: namespaceCfgSynced,
		},
		{
			name:   "unmanage if invalid",
			actual: deployment(appsv1.RecreateDeploymentStrategyType, syncertesting.ManagementInvalid),
			expectUpdate: &syncertesting.Diff{
				Declared: deployment(appsv1.RecreateDeploymentStrategyType),
				Actual:   deployment(appsv1.RecreateDeploymentStrategyType, syncertesting.ManagementInvalid),
			},
			expectStatusUpdate: namespaceCfgSynced,
			expectEvent:        managedNamespaceReconcileComplete,
		},
	}

	toSync := []schema.GroupVersionKind{kinds.Deployment()}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			tm := syncertesting.NewTestMocks(t, mockCtrl)

			fakeDecoder := mocks.NewFakeDecoder(syncertesting.ToUnstructuredList(t, syncertesting.Converter, tc.declared))
			testReconciler := NewNamespaceConfigReconciler(ctx,
				client.New(tm.MockClient), tm.MockApplier, tm.MockCache, tm.MockRecorder, fakeDecoder, syncertesting.Now, toSync)

			tm.ExpectNamespaceCacheGet(namespaceCfg, managedNamespace)

			tm.ExpectNamespaceConfigClientGet(namespaceCfg)
			tm.ExpectNamespaceClientGet(unmanagedNamespace)

			tm.ExpectNamespaceUpdate(managedNamespace)

			tm.ExpectCacheList(kinds.Deployment(), managedNamespace.Name, tc.actual)
			tm.ExpectCreate(tc.expectCreate)
			tm.ExpectUpdate(tc.expectUpdate)
			tm.ExpectDelete(tc.expectDelete)

			tm.ExpectNamespaceStatusUpdate(tc.expectStatusUpdate)
			tm.ExpectEvent(tc.expectEvent)

			_, err := testReconciler.Reconcile(
				reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name: managedNamespace.Name,
					},
				})
			if err != nil {
				t.Errorf("unexpected reconciliation error: %v", err)
			}
		})
	}
}

func TestUnmanagedNamespaceReconcile(t *testing.T) {
	testCases := []struct {
		name                string
		namespaceConfig     *v1.NamespaceConfig
		namespace           *corev1.Namespace
		wantNamespaceUpdate *corev1.Namespace
		wantStatusUpdate    *v1.NamespaceConfig
		declared            runtime.Object
		actual              runtime.Object
		wantUpdate          *syncertesting.Diff
		wantEvent           *syncertesting.Event
	}{
		{
			name:                "clean up unmanaged namespace with namespaceconfig",
			namespaceConfig:     namespaceConfig("eng", v1.StateSynced, syncertesting.ImportToken(syncertesting.Token), syncertesting.SyncToken(), syncertesting.ManagementDisabled),
			namespace:           namespace("eng", managedQuotaLabels, syncertesting.ManagementEnabled),
			wantNamespaceUpdate: namespace("eng"),
			wantStatusUpdate: namespaceConfig("eng", v1.StateError, syncertesting.ImportToken(syncertesting.Token), syncertesting.SyncTime(), syncertesting.SyncToken(),
				namespaceSyncError(v1.ConfigManagementError{
					ResourceName: "eng",
					ResourceGVK:  corev1.SchemeGroupVersion.WithKind("Namespace"),
					ErrorMessage: unmanagedError(),
				}), syncertesting.ManagementDisabled),
			wantEvent: &syncertesting.Event{
				Kind:   corev1.EventTypeWarning,
				Reason: "UnmanagedNamespace",
				Obj:    namespace("eng", managedQuotaLabels, syncertesting.ManagementEnabled),
			},
		},
		{
			name:                "clean up unmanaged namespace without namespaceconfig",
			namespace:           namespace("eng", managedQuotaLabels),
			wantNamespaceUpdate: namespace("eng"),
		},
		{
			name:            "unmanaged namespace has resources synced but status error",
			namespaceConfig: namespaceConfig("eng", v1.StateSynced, syncertesting.ImportToken(syncertesting.Token), syncertesting.ManagementDisabled),
			namespace:       namespace("eng"),
			declared:        deployment(appsv1.RecreateDeploymentStrategyType),
			actual:          deployment(appsv1.RollingUpdateDeploymentStrategyType, object.Namespace("eng"), syncertesting.TokenAnnotation),
			wantStatusUpdate: namespaceConfig("eng", v1.StateError, syncertesting.ImportToken(syncertesting.Token), syncertesting.SyncTime(), syncertesting.SyncToken(),
				namespaceSyncError(v1.ConfigManagementError{
					ResourceName: "eng",
					ResourceGVK:  corev1.SchemeGroupVersion.WithKind("Namespace"),
					ErrorMessage: unmanagedError(),
				}), syncertesting.ManagementDisabled),
			wantEvent: &syncertesting.Event{
				Kind:   corev1.EventTypeWarning,
				Reason: "UnmanagedNamespace",
				Obj:    namespace("eng"),
			},
			wantUpdate: &syncertesting.Diff{
				Declared: deployment(appsv1.RecreateDeploymentStrategyType, object.Namespace("eng"), syncertesting.ManagementEnabled, syncertesting.TokenAnnotation),
				Actual:   deployment(appsv1.RollingUpdateDeploymentStrategyType, object.Namespace("eng"), syncertesting.TokenAnnotation),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			toSync := []schema.GroupVersionKind{kinds.Deployment()}

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			tm := syncertesting.NewTestMocks(t, mockCtrl)
			fakeDecoder := mocks.NewFakeDecoder(syncertesting.ToUnstructuredList(t, syncertesting.Converter, tc.declared))
			testReconciler := NewNamespaceConfigReconciler(ctx,
				client.New(tm.MockClient), tm.MockApplier, tm.MockCache, tm.MockRecorder, fakeDecoder, syncertesting.Now, toSync)

			tm.ExpectNamespaceCacheGet(tc.namespaceConfig, tc.namespace)

			tm.ExpectNamespaceConfigClientGet(tc.namespaceConfig)
			if tc.wantNamespaceUpdate != nil {
				tm.ExpectNamespaceClientGet(tc.namespace)
			}

			tm.ExpectUpdate(tc.wantUpdate)
			tm.ExpectNamespaceUpdate(tc.wantNamespaceUpdate)

			if tc.namespaceConfig != nil {
				tm.ExpectCacheList(kinds.Deployment(), tc.namespace.Name, tc.actual)
			}

			tm.ExpectEvent(tc.wantEvent)
			tm.ExpectNamespaceStatusUpdate(tc.wantStatusUpdate)

			_, err := testReconciler.Reconcile(
				reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name: tc.namespace.Name,
					},
				})
			if err != nil {
				t.Errorf("unexpected reconciliation error: %v", err)
			}
		})
	}
}

func TestSpecialNamespaceReconcile(t *testing.T) {
	testCases := []struct {
		name                string
		namespaceConfig     *v1.NamespaceConfig
		namespace           *corev1.Namespace
		wantNamespaceUpdate *corev1.Namespace
		wantStatusUpdate    *v1.NamespaceConfig
	}{
		{
			name:                "do not add quota enforcement label on managed kube-system",
			namespaceConfig:     namespaceConfig("kube-system", v1.StateSynced, syncertesting.ImportToken(syncertesting.Token)),
			namespace:           namespace("kube-system", syncertesting.ManagementEnabled),
			wantNamespaceUpdate: namespace("kube-system", syncertesting.ManagementEnabled, syncertesting.TokenAnnotation),
			wantStatusUpdate: namespaceConfig("kube-system", v1.StateSynced, syncertesting.ImportToken(syncertesting.Token),
				syncertesting.SyncTime(), syncertesting.SyncToken()),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			toSync := []schema.GroupVersionKind{kinds.Deployment()}

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			tm := syncertesting.NewTestMocks(t, mockCtrl)
			fakeDecoder := mocks.NewFakeDecoder(syncertesting.ToUnstructuredList(t, syncertesting.Converter, nil))
			testReconciler := NewNamespaceConfigReconciler(ctx,
				client.New(tm.MockClient), tm.MockApplier, tm.MockCache, tm.MockRecorder, fakeDecoder, syncertesting.Now, toSync)

			tm.ExpectNamespaceCacheGet(tc.namespaceConfig, tc.namespace)

			tm.ExpectNamespaceConfigClientGet(tc.namespaceConfig)
			tm.ExpectNamespaceClientGet(unmanaged(tc.namespace))

			tm.ExpectNamespaceUpdate(tc.wantNamespaceUpdate)

			tm.ExpectCacheList(kinds.Deployment(), tc.namespace.Name, nil)

			tm.ExpectNamespaceStatusUpdate(tc.wantStatusUpdate)

			_, err := testReconciler.Reconcile(
				reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name: tc.namespace.Name,
					},
				})
			if err != nil {
				t.Errorf("unexpected reconciliation error: %v", err)
			}
		})
	}
}

func TestNamespaceConfigReconcile(t *testing.T) {
	testCases := []struct {
		name string
		// What the namespace resource looks like on the cluster before the
		// reconcile. Set to nil if the namespace is not present.
		namespace *corev1.Namespace
		// The objects present in the corresponding namespace on the cluster.
		actual []runtime.Object
		// What the namespace should look like after an update.
		wantNamespaceUpdate *corev1.Namespace
		// The objects that got deleted in the namespace.
		wantDelete runtime.Object
		// This is what the NamespaceConfig resource should look like (with
		// updated Status field) on the cluster.
		wantStatusUpdate *v1.NamespaceConfig
		// The events that are expected to be emitted as the result of the
		// operation.
		wantEvent *syncertesting.Event
		// By default all tests only sync Deployments.
		// Specify this to override the resources being synced.
		toSyncOverride schema.GroupVersionKind
	}{
		{
			name: "default namespace is not deleted when namespace config is removed",
			namespace: namespace("default", object.Annotations(
				map[string]string{
					v1.ClusterNameAnnotationKey:       "cluster-name",
					v1.ClusterSelectorAnnotationKey:   "some-selector",
					v1.NamespaceSelectorAnnotationKey: "some-selector",
					v1.ResourceManagementKey:          v1.ResourceManagementEnabled,
					v1.SourcePathAnnotationKey:        "some-path",
					v1.SyncTokenAnnotationKey:         "syncertesting.Token",
					"some-user-annotation":            "some-annotation-value",
				},
			),
				object.Labels(
					map[string]string{
						v1.ConfigManagementQuotaKey:  "some-quota",
						v1.ConfigManagementSystemKey: "not-used",
						"some-user-label":            "some-label-value",
					},
				),
			),
			wantNamespaceUpdate: namespace("default",
				object.Annotation("some-user-annotation", "some-annotation-value"),
				object.Label(v1.ConfigManagementSystemKey, "not-used"),
				object.Label("some-user-label", "some-label-value"),
			),
			actual: []runtime.Object{
				deployment(
					appsv1.RecreateDeploymentStrategyType,
					object.Namespace("default"),
					syncertesting.ManagementEnabled,
					object.Name("my-deployment")),
				deployment(appsv1.RecreateDeploymentStrategyType,
					object.Namespace("default"),
					object.Name("your-deployment")),
			},
			wantDelete: deployment(
				appsv1.RecreateDeploymentStrategyType,
				object.Namespace("default"),
				syncertesting.ManagementEnabled,
				object.Name("my-deployment")),
			wantEvent: &syncertesting.Event{
				Kind:    corev1.EventTypeNormal,
				Reason:  "ReconcileComplete",
				Varargs: true,
				Obj:     &v1.NamespaceConfig{},
			},
		},
		{
			name: "kube-system namespace is not deleted when namespace config is removed",
			namespace: namespace("kube-system", object.Annotations(
				map[string]string{
					v1.ClusterNameAnnotationKey:       "cluster-name",
					v1.ClusterSelectorAnnotationKey:   "some-selector",
					v1.NamespaceSelectorAnnotationKey: "some-selector",
					v1.ResourceManagementKey:          v1.ResourceManagementEnabled,
					v1.SourcePathAnnotationKey:        "some-path",
					v1.SyncTokenAnnotationKey:         "syncertesting.Token",
					"some-user-annotation":            "some-annotation-value",
				},
			),
				object.Labels(
					map[string]string{
						v1.ConfigManagementQuotaKey:  "some-quota",
						v1.ConfigManagementSystemKey: "not-used",
						"some-user-label":            "some-label-value",
					},
				),
			),
			wantNamespaceUpdate: namespace("kube-system",
				object.Annotation("some-user-annotation", "some-annotation-value"),
				object.Label(v1.ConfigManagementSystemKey, "not-used"),
				object.Label("some-user-label", "some-label-value"),
			),
			actual: []runtime.Object{
				deployment(
					appsv1.RecreateDeploymentStrategyType,
					object.Namespace("kube-system"),
					syncertesting.ManagementEnabled,
					object.Name("my-deployment")),
				deployment(appsv1.RecreateDeploymentStrategyType,
					object.Namespace("kube-system"),
					object.Name("your-deployment")),
			},
			wantDelete: deployment(
				appsv1.RecreateDeploymentStrategyType,
				object.Namespace("kube-system"),
				syncertesting.ManagementEnabled,
				object.Name("my-deployment")),
			wantEvent: &syncertesting.Event{
				Kind:    corev1.EventTypeNormal,
				Reason:  "ReconcileComplete",
				Varargs: true,
				Obj:     &v1.NamespaceConfig{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			toSync := []schema.GroupVersionKind{kinds.Deployment()}
			if !tc.toSyncOverride.Empty() {
				toSync = []schema.GroupVersionKind{tc.toSyncOverride}
			}

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			tm := syncertesting.NewTestMocks(t, mockCtrl)
			fakeDecoder := mocks.NewFakeDecoder(syncertesting.ToUnstructuredList(t, syncertesting.Converter, nil))
			testReconciler := NewNamespaceConfigReconciler(ctx,
				client.New(tm.MockClient), tm.MockApplier, tm.MockCache, tm.MockRecorder, fakeDecoder, syncertesting.Now, toSync)

			tm.ExpectNamespaceCacheGet(nil, tc.namespace)
			tm.ExpectNamespaceClientGet(tc.namespace)

			tm.ExpectNamespaceUpdate(tc.wantNamespaceUpdate)

			tm.ExpectCacheList(kinds.Deployment(), tc.namespace.Name, tc.actual...)
			tm.ExpectDelete(tc.wantDelete)

			tm.ExpectEvent(tc.wantEvent)

			tm.ExpectNamespaceStatusUpdate(tc.wantStatusUpdate)

			_, err := testReconciler.Reconcile(
				reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name: tc.namespace.Name,
					},
				})
			if err != nil {
				t.Errorf("unexpected reconciliation error: %v", err)
			}
		})
	}
}

func unmanaged(o *corev1.Namespace) *corev1.Namespace {
	r := o.DeepCopy()
	object.RemoveAnnotations(r, v1.ResourceManagementKey)
	object.RemoveLabels(r, v1.ManagedByKey)
	return r
}
