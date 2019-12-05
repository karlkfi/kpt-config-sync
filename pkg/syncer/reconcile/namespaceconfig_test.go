package reconcile

import (
	"context"
	"testing"

	"github.com/google/nomos/pkg/syncer/metrics"

	"github.com/golang/mock/gomock"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
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

func deployment(deploymentStrategy appsv1.DeploymentStrategyType, opts ...core.MetaMutator) *appsv1.Deployment {
	mutators := append(opts, syncertesting.Herrings...)
	result := fake.DeploymentObject(mutators...)
	result.Spec.Strategy.Type = deploymentStrategy
	return result
}

func namespace(name string, opts ...core.MetaMutator) *corev1.Namespace {
	return fake.NamespaceObject(name, opts...)
}

func namespaceConfig(name string, state v1.ConfigSyncState, opts ...fake.NamespaceConfigMutator) *v1.NamespaceConfig {
	result := fake.NamespaceConfigObject(opts...)
	result.Name = name
	result.Status.SyncState = state
	return result
}

func namespaceSyncError(err v1.ConfigManagementError) fake.NamespaceConfigMutator {
	return func(nc *v1.NamespaceConfig) {
		nc.Status.SyncErrors = append(nc.Status.SyncErrors, err)
	}
}

var (
	eng                = "eng"
	managedNamespace   = namespace(eng, syncertesting.ManagementEnabled, syncertesting.TokenAnnotation)
	unmanagedNamespace = namespace(eng)

	namespaceCfg       = namespaceConfig(eng, v1.StateSynced, syncertesting.NamespaceConfigImportToken(syncertesting.Token))
	namespaceCfgSynced = namespaceConfig(eng, v1.StateSynced, syncertesting.NamespaceConfigImportToken(syncertesting.Token),
		syncertesting.NamespaceConfigSyncTime(), syncertesting.NamespaceConfigSyncToken())

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
			expectCreate:       deployment(appsv1.RollingUpdateDeploymentStrategyType, syncertesting.ManagementEnabled, syncertesting.TokenAnnotation),
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
					syncertesting.ManagementInvalid, syncertesting.TokenAnnotation)),
			},
		},
		{
			name:     "update to declared state",
			declared: deployment(appsv1.RollingUpdateDeploymentStrategyType),
			actual:   deployment(appsv1.RecreateDeploymentStrategyType, syncertesting.ManagementEnabled),
			expectUpdate: &syncertesting.Diff{
				Declared: deployment(appsv1.RollingUpdateDeploymentStrategyType, syncertesting.ManagementEnabled, syncertesting.TokenAnnotation),
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
				Declared: deployment(appsv1.RollingUpdateDeploymentStrategyType, syncertesting.ManagementEnabled, syncertesting.TokenAnnotation),
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
				Declared: deployment(appsv1.RollingUpdateDeploymentStrategyType, syncertesting.ManagementEnabled, syncertesting.TokenAnnotation),
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
					syncertesting.ManagementInvalid, syncertesting.TokenAnnotation)),
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
				client.New(tm.MockClient, metrics.APICallDuration), tm.MockApplier, tm.MockCache, tm.MockRecorder, fakeDecoder, syncertesting.Now, toSync)

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
			namespaceConfig:     namespaceConfig("eng", v1.StateSynced, syncertesting.NamespaceConfigImportToken(syncertesting.Token), syncertesting.NamespaceConfigSyncToken(), fake.NamespaceConfigMeta(syncertesting.ManagementDisabled)),
			namespace:           namespace("eng", syncertesting.ManagementEnabled),
			wantNamespaceUpdate: namespace("eng"),
			wantStatusUpdate: namespaceConfig("eng", v1.StateError, syncertesting.NamespaceConfigImportToken(syncertesting.Token), syncertesting.NamespaceConfigSyncTime(), syncertesting.NamespaceConfigSyncToken(),
				namespaceSyncError(v1.ConfigManagementError{
					ErrorResources: []v1.ErrorResource{
						{
							ResourceName: "eng",
							ResourceGVK:  corev1.SchemeGroupVersion.WithKind("Namespace"),
						},
					},
					ErrorMessage: unmanagedError(),
				}), fake.NamespaceConfigMeta(syncertesting.ManagementDisabled)),
			wantEvent: &syncertesting.Event{
				Kind:   corev1.EventTypeWarning,
				Reason: "UnmanagedNamespace",
				Obj:    namespace("eng", syncertesting.ManagementEnabled),
			},
		},
		{
			name:            "unmanaged namespace has resources synced but status error",
			namespaceConfig: namespaceConfig("eng", v1.StateSynced, syncertesting.NamespaceConfigImportToken(syncertesting.Token), fake.NamespaceConfigMeta(syncertesting.ManagementDisabled)),
			namespace:       namespace("eng"),
			declared:        deployment(appsv1.RecreateDeploymentStrategyType),
			actual:          deployment(appsv1.RollingUpdateDeploymentStrategyType, syncertesting.TokenAnnotation),
			wantStatusUpdate: namespaceConfig("eng", v1.StateError, syncertesting.NamespaceConfigImportToken(syncertesting.Token), syncertesting.NamespaceConfigSyncTime(), syncertesting.NamespaceConfigSyncToken(),
				namespaceSyncError(v1.ConfigManagementError{
					ErrorResources: []v1.ErrorResource{
						{
							ResourceName: "eng",
							ResourceGVK:  corev1.SchemeGroupVersion.WithKind("Namespace"),
						},
					},
					ErrorMessage: unmanagedError(),
				}), fake.NamespaceConfigMeta(syncertesting.ManagementDisabled)),
			wantEvent: &syncertesting.Event{
				Kind:   corev1.EventTypeWarning,
				Reason: "UnmanagedNamespace",
				Obj:    namespace("eng"),
			},
			wantUpdate: &syncertesting.Diff{
				Declared: deployment(appsv1.RecreateDeploymentStrategyType, syncertesting.ManagementEnabled, syncertesting.TokenAnnotation),
				Actual:   deployment(appsv1.RollingUpdateDeploymentStrategyType, syncertesting.TokenAnnotation),
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
				client.New(tm.MockClient, metrics.APICallDuration), tm.MockApplier, tm.MockCache, tm.MockRecorder, fakeDecoder, syncertesting.Now, toSync)

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
			namespaceConfig:     namespaceConfig("kube-system", v1.StateSynced, syncertesting.NamespaceConfigImportToken(syncertesting.Token)),
			namespace:           namespace("kube-system", syncertesting.ManagementEnabled),
			wantNamespaceUpdate: namespace("kube-system", syncertesting.ManagementEnabled, syncertesting.TokenAnnotation),
			wantStatusUpdate: namespaceConfig("kube-system", v1.StateSynced, syncertesting.NamespaceConfigImportToken(syncertesting.Token),
				syncertesting.NamespaceConfigSyncTime(), syncertesting.NamespaceConfigSyncToken()),
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
				client.New(tm.MockClient, metrics.APICallDuration), tm.MockApplier, tm.MockCache, tm.MockRecorder, fakeDecoder, syncertesting.Now, toSync)

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
		namespace       *corev1.Namespace
		namespaceConfig *v1.NamespaceConfig
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
			namespace: namespace("default", core.Annotations(
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
				core.Labels(
					map[string]string{
						"some-user-label": "some-label-value",
					},
				),
			),
			namespaceConfig: namespaceConfig("default",
				v1.StateSynced,
				syncertesting.NamespaceConfigImportToken(syncertesting.Token),
				syncertesting.MarkForDeletion(),
			),
			wantNamespaceUpdate: namespace("default",
				core.Annotation("some-user-annotation", "some-annotation-value"),
				core.Label("some-user-label", "some-label-value"),
			),
			actual: []runtime.Object{
				deployment(
					appsv1.RecreateDeploymentStrategyType,
					core.Namespace("default"),
					syncertesting.ManagementEnabled,
					core.Name("my-deployment")),
				deployment(appsv1.RecreateDeploymentStrategyType,
					core.Namespace("default"),
					core.Name("your-deployment")),
			},
			wantDelete: deployment(
				appsv1.RecreateDeploymentStrategyType,
				core.Namespace("default"),
				syncertesting.ManagementEnabled,
				core.Name("my-deployment")),
			wantEvent: &syncertesting.Event{
				Kind:    corev1.EventTypeNormal,
				Reason:  "ReconcileComplete",
				Varargs: true,
				Obj:     &v1.NamespaceConfig{},
			},
		},
		{
			name: "kube-system namespace is not deleted when namespace config is removed",
			namespace: namespace("kube-system", core.Annotations(
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
				core.Labels(
					map[string]string{
						"some-user-label": "some-label-value",
					},
				),
			),
			namespaceConfig: namespaceConfig("kube-system",
				v1.StateSynced,
				syncertesting.NamespaceConfigImportToken(syncertesting.Token),
				syncertesting.MarkForDeletion(),
			),
			wantNamespaceUpdate: namespace("kube-system",
				core.Annotation("some-user-annotation", "some-annotation-value"),
				core.Label("some-user-label", "some-label-value"),
			),
			actual: []runtime.Object{
				deployment(
					appsv1.RecreateDeploymentStrategyType,
					core.Namespace("kube-system"),
					syncertesting.ManagementEnabled,
					core.Name("my-deployment")),
				deployment(appsv1.RecreateDeploymentStrategyType,
					core.Namespace("kube-system"),
					core.Name("your-deployment")),
			},
			wantDelete: deployment(
				appsv1.RecreateDeploymentStrategyType,
				core.Namespace("kube-system"),
				syncertesting.ManagementEnabled,
				core.Name("my-deployment")),
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
				client.New(tm.MockClient, metrics.APICallDuration), tm.MockApplier, tm.MockCache, tm.MockRecorder, fakeDecoder, syncertesting.Now, toSync)

			tm.ExpectNamespaceCacheGet(tc.namespaceConfig, tc.namespace)

			tm.ExpectNamespaceClientGet(tc.namespace)

			tm.ExpectNamespaceConfigClientGet(tc.namespaceConfig)

			tm.ExpectNamespaceUpdate(tc.wantNamespaceUpdate)

			tm.ExpectCacheList(kinds.Deployment(), tc.namespace.Name, tc.actual...)
			tm.ExpectNamespaceConfigDelete(tc.namespaceConfig)
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
	core.RemoveAnnotations(r, v1.ResourceManagementKey)
	core.RemoveLabels(r, v1.ManagedByKey)
	return r
}
