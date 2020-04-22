package reconcile_test

import (
	"context"
	"testing"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/metrics"
	syncerreconcile "github.com/google/nomos/pkg/syncer/reconcile"
	syncertesting "github.com/google/nomos/pkg/syncer/testing"
	testingfake "github.com/google/nomos/pkg/syncer/testing/fake"
	"github.com/google/nomos/pkg/testing/fake"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func deployment(deploymentStrategy appsv1.DeploymentStrategyType, opts ...core.MetaMutator) *appsv1.Deployment {
	mutators := append([]core.MetaMutator{core.Namespace(eng)})
	mutators = append(mutators, opts...)
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

var (
	eng              = "eng"
	managedNamespace = namespace(eng, syncertesting.ManagementEnabled, syncertesting.TokenAnnotation)

	namespaceCfg       = namespaceConfig(eng, v1.StateSynced, syncertesting.NamespaceConfigImportToken(syncertesting.Token))
	namespaceCfgSynced = namespaceConfig(eng, v1.StateSynced, syncertesting.NamespaceConfigImportToken(syncertesting.Token),
		syncertesting.NamespaceConfigSyncTime(), syncertesting.NamespaceConfigSyncToken())

	managedNamespaceReconcileComplete = testingfake.NewEvent(namespaceCfg, corev1.EventTypeNormal, v1.EventReasonReconcileComplete)
)

func TestManagedNamespaceConfigReconcile(t *testing.T) {
	testCases := []struct {
		name      string
		declared  runtime.Object
		actual    runtime.Object
		want      []runtime.Object
		wantEvent *testingfake.Event
	}{
		{
			name:     "create from declared state",
			declared: deployment(appsv1.RollingUpdateDeploymentStrategyType),
			want: []runtime.Object{
				namespaceCfgSynced,
				deployment(appsv1.RollingUpdateDeploymentStrategyType, syncertesting.ManagementEnabled, syncertesting.TokenAnnotation),
			},
			wantEvent: managedNamespaceReconcileComplete,
		},
		{
			name:     "do not create if management disabled",
			declared: deployment(appsv1.RollingUpdateDeploymentStrategyType, syncertesting.ManagementDisabled),
			want: []runtime.Object{
				namespaceCfgSynced,
			},
		},
		{
			name:     "do not create if management invalid",
			declared: deployment(appsv1.RollingUpdateDeploymentStrategyType, syncertesting.ManagementInvalid),
			want: []runtime.Object{
				namespaceCfgSynced,
			},
			wantEvent: testingfake.NewEvent(
				deployment(appsv1.RollingUpdateDeploymentStrategyType),
				corev1.EventTypeWarning, v1.EventReasonInvalidAnnotation),
		},
		{
			name:     "update to declared state",
			declared: deployment(appsv1.RollingUpdateDeploymentStrategyType),
			actual:   deployment(appsv1.RecreateDeploymentStrategyType, syncertesting.ManagementEnabled),
			want: []runtime.Object{
				namespaceCfgSynced,
				deployment(appsv1.RollingUpdateDeploymentStrategyType, syncertesting.ManagementEnabled, syncertesting.TokenAnnotation),
			},
			wantEvent: managedNamespaceReconcileComplete,
		},
		{
			name:     "update to declared state even if actual managed unset",
			declared: deployment(appsv1.RollingUpdateDeploymentStrategyType),
			actual:   deployment(appsv1.RecreateDeploymentStrategyType),
			want: []runtime.Object{
				namespaceCfgSynced,
				deployment(appsv1.RollingUpdateDeploymentStrategyType, syncertesting.ManagementEnabled, syncertesting.TokenAnnotation),
			},
			wantEvent: managedNamespaceReconcileComplete,
		},
		{
			name:     "update to declared state even if actual managed invalid",
			declared: deployment(appsv1.RollingUpdateDeploymentStrategyType),
			actual:   deployment(appsv1.RecreateDeploymentStrategyType, syncertesting.ManagementInvalid),
			want: []runtime.Object{
				namespaceCfgSynced,
				deployment(appsv1.RollingUpdateDeploymentStrategyType, syncertesting.ManagementEnabled, syncertesting.TokenAnnotation),
			},
			wantEvent: managedNamespaceReconcileComplete,
		},
		{
			name:     "do not update if declared managed invalid",
			declared: deployment(appsv1.RollingUpdateDeploymentStrategyType, syncertesting.ManagementInvalid),
			actual:   deployment(appsv1.RecreateDeploymentStrategyType),
			want: []runtime.Object{
				namespaceCfgSynced,
				deployment(appsv1.RecreateDeploymentStrategyType),
			},
			wantEvent: testingfake.NewEvent(
				deployment(appsv1.RollingUpdateDeploymentStrategyType),
				corev1.EventTypeWarning, v1.EventReasonInvalidAnnotation),
		},
		{
			name:     "update to unmanaged",
			declared: deployment(appsv1.RollingUpdateDeploymentStrategyType, syncertesting.ManagementDisabled),
			actual:   deployment(appsv1.RecreateDeploymentStrategyType, syncertesting.ManagementEnabled),
			want: []runtime.Object{
				namespaceCfgSynced,
				deployment(appsv1.RecreateDeploymentStrategyType),
			},
			wantEvent: managedNamespaceReconcileComplete,
		},
		{
			name:     "do not update if unmanaged",
			declared: deployment(appsv1.RollingUpdateDeploymentStrategyType, syncertesting.ManagementDisabled),
			actual:   deployment(appsv1.RecreateDeploymentStrategyType),
			want: []runtime.Object{
				namespaceCfgSynced,
				deployment(appsv1.RecreateDeploymentStrategyType),
			},
		},
		{
			name:   "delete if managed",
			actual: deployment(appsv1.RecreateDeploymentStrategyType, syncertesting.ManagementEnabled),
			want: []runtime.Object{
				namespaceCfgSynced,
			},
			wantEvent: managedNamespaceReconcileComplete,
		},
		{
			name:   "do not delete if unmanaged",
			actual: deployment(appsv1.RecreateDeploymentStrategyType),
			want: []runtime.Object{
				namespaceCfgSynced,
				deployment(appsv1.RecreateDeploymentStrategyType),
			},
		},
		{
			name:   "unmanage if invalid",
			actual: deployment(appsv1.RecreateDeploymentStrategyType, syncertesting.ManagementInvalid),
			want: []runtime.Object{
				namespaceCfgSynced,
				deployment(appsv1.RecreateDeploymentStrategyType),
			},
			wantEvent: managedNamespaceReconcileComplete,
		},
	}

	toSync := []schema.GroupVersionKind{kinds.Deployment()}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			fakeDecoder := testingfake.NewDecoder(syncertesting.ToUnstructuredList(t, syncertesting.Converter, tc.declared))
			fakeEventRecorder := testingfake.NewEventRecorder(t)
			actual := []runtime.Object{namespaceCfg, managedNamespace}
			if tc.actual != nil {
				actual = append(actual, tc.actual)
			}
			s := runtime.NewScheme()
			s.AddKnownTypeWithName(kinds.Namespace(), &corev1.Namespace{})
			s.AddKnownTypeWithName(kinds.Deployment(), &appsv1.Deployment{})
			fakeClient := testingfake.NewClient(t, s, actual...)

			testReconciler := syncerreconcile.NewNamespaceConfigReconciler(ctx,
				client.New(fakeClient, metrics.APICallDuration), fakeClient.Applier(), fakeClient, fakeEventRecorder, fakeDecoder, syncertesting.Now, toSync)

			_, err := testReconciler.Reconcile(
				reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name: managedNamespace.Name,
					},
				})
			if err != nil {
				t.Errorf("unexpected reconciliation error: %v", err)
			}

			if tc.wantEvent != nil {
				fakeEventRecorder.Check(t, *tc.wantEvent)
			} else {
				fakeEventRecorder.Check(t)
			}
			want := append([]runtime.Object{managedNamespace}, tc.want...)
			fakeClient.Check(t, want...)
		})
	}
}

func TestUnmanagedNamespaceReconcile(t *testing.T) {
	testCases := []struct {
		name                   string
		actualNamespaceConfig  *v1.NamespaceConfig
		namespace              *corev1.Namespace
		wantNamespace          *corev1.Namespace
		updatedNamespaceConfig *v1.NamespaceConfig
		declared               runtime.Object
		actual                 runtime.Object
		wantEvent              *testingfake.Event
		want                   runtime.Object
	}{
		{
			name:                  "clean up unmanaged Namespace with namespaceconfig",
			actualNamespaceConfig: namespaceConfig("eng", v1.StateSynced, syncertesting.NamespaceConfigImportToken(syncertesting.Token), syncertesting.NamespaceConfigSyncToken(), fake.NamespaceConfigMeta(syncertesting.ManagementDisabled)),
			namespace:             namespace("eng", syncertesting.ManagementEnabled),
			wantNamespace:         namespace("eng"),
		},
		{
			name:                  "do nothing to explicitly unmanaged resources",
			actualNamespaceConfig: namespaceConfig("eng", v1.StateSynced, syncertesting.NamespaceConfigImportToken(syncertesting.Token), syncertesting.NamespaceConfigSyncToken(), fake.NamespaceConfigMeta(syncertesting.ManagementDisabled, core.Label("not", "synced"))),
			namespace:             namespace("eng"),
			declared:              deployment(appsv1.RollingUpdateDeploymentStrategyType, syncertesting.ManagementDisabled, core.Label("also not", "synced")),
			actual:                deployment(appsv1.RecreateDeploymentStrategyType),
			wantNamespace:         namespace("eng"),
			want:                  deployment(appsv1.RecreateDeploymentStrategyType),
		},
		{
			name:                   "delete resources in unmanaged Namespace",
			actualNamespaceConfig:  namespaceConfig("eng", v1.StateSynced, fake.NamespaceConfigMeta(syncertesting.ManagementDisabled)),
			namespace:              namespace("eng"),
			updatedNamespaceConfig: namespaceConfig("eng", v1.StateSynced, fake.NamespaceConfigMeta(syncertesting.ManagementDisabled)),
			actual:                 deployment(appsv1.RollingUpdateDeploymentStrategyType, syncertesting.ManagementEnabled, syncertesting.TokenAnnotation),
			wantEvent: testingfake.NewEvent(namespaceConfig("eng", v1.StateSynced),
				corev1.EventTypeNormal, v1.EventReasonReconcileComplete),
			wantNamespace: namespace("eng"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			toSync := []schema.GroupVersionKind{kinds.Deployment()}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			fakeDecoder := testingfake.NewDecoder(syncertesting.ToUnstructuredList(t, syncertesting.Converter))
			fakeEventRecorder := testingfake.NewEventRecorder(t)
			s := runtime.NewScheme()
			s.AddKnownTypeWithName(kinds.Namespace(), &corev1.Namespace{})
			actual := []runtime.Object{tc.actualNamespaceConfig, tc.namespace}
			if tc.actual != nil {
				actual = append(actual, tc.actual)
			}
			fakeClient := testingfake.NewClient(t, s, actual...)

			testReconciler := syncerreconcile.NewNamespaceConfigReconciler(ctx,
				client.New(fakeClient, metrics.APICallDuration), fakeClient.Applier(), fakeClient, fakeEventRecorder, fakeDecoder, syncertesting.Now, toSync)

			_, err := testReconciler.Reconcile(
				reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name: tc.namespace.Name,
					},
				})

			if err != nil {
				t.Errorf("unexpected reconciliation error: %v", err)
			}

			want := []runtime.Object{tc.wantNamespace}
			if tc.updatedNamespaceConfig != nil {
				want = append(want, tc.updatedNamespaceConfig)
			} else {
				want = append(want, tc.actualNamespaceConfig)
			}
			if tc.want != nil {
				want = append(want, tc.want)
			}
			fakeClient.Check(t, want...)

			if tc.wantEvent != nil {
				fakeEventRecorder.Check(t, *tc.wantEvent)
			} else {
				fakeEventRecorder.Check(t)
			}
		})
	}
}

func TestSpecialNamespaceReconcile(t *testing.T) {
	testCases := []struct {
		name          string
		declared      *v1.NamespaceConfig
		actual        *corev1.Namespace
		wantNamespace *corev1.Namespace
		want          *v1.NamespaceConfig
	}{
		{
			name:          "do not add quota enforcement label on managed kube-system",
			declared:      namespaceConfig("kube-system", v1.StateSynced, syncertesting.NamespaceConfigImportToken(syncertesting.Token)),
			actual:        namespace("kube-system", syncertesting.ManagementEnabled),
			wantNamespace: namespace("kube-system", syncertesting.ManagementEnabled, syncertesting.TokenAnnotation),
			want: namespaceConfig("kube-system", v1.StateSynced, syncertesting.NamespaceConfigImportToken(syncertesting.Token),
				syncertesting.NamespaceConfigSyncTime(), syncertesting.NamespaceConfigSyncToken()),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			toSync := []schema.GroupVersionKind{kinds.Deployment()}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			fakeDecoder := testingfake.NewDecoder(syncertesting.ToUnstructuredList(t, syncertesting.Converter, nil))
			fakeEventRecorder := testingfake.NewEventRecorder(t)
			s := runtime.NewScheme()
			s.AddKnownTypeWithName(kinds.Namespace(), &corev1.Namespace{})
			fakeClient := testingfake.NewClient(t, s, tc.declared, tc.actual)

			testReconciler := syncerreconcile.NewNamespaceConfigReconciler(ctx,
				client.New(fakeClient, metrics.APICallDuration), fakeClient.Applier(), fakeClient, fakeEventRecorder, fakeDecoder, syncertesting.Now, toSync)

			_, err := testReconciler.Reconcile(
				reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name: tc.actual.Name,
					},
				})

			if err != nil {
				t.Errorf("unexpected reconciliation error: %v", err)
			}

			fakeEventRecorder.Check(t)
			fakeClient.Check(t, tc.want, tc.wantNamespace)
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
		// want is the objects on the API Server after reconciliation.
		want []runtime.Object
		// The events that are expected to be emitted as the result of the
		// operation.
		wantEvent *testingfake.Event
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
			want: []runtime.Object{
				namespace("default",
					core.Annotation("some-user-annotation", "some-annotation-value"),
					core.Label("some-user-label", "some-label-value"),
				),
				deployment(appsv1.RecreateDeploymentStrategyType,
					core.Namespace("default"),
					core.Name("your-deployment")),
			},
			wantEvent: testingfake.NewEvent(namespaceConfig("", v1.StateUnknown),
				corev1.EventTypeNormal, v1.EventReasonReconcileComplete),
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
			want: []runtime.Object{
				namespace("kube-system",
					core.Annotation("some-user-annotation", "some-annotation-value"),
					core.Label("some-user-label", "some-label-value"),
				),
				deployment(appsv1.RecreateDeploymentStrategyType,
					core.Namespace("kube-system"),
					core.Name("your-deployment")),
			},
			wantEvent: testingfake.NewEvent(namespaceConfig("", v1.StateUnknown),
				corev1.EventTypeNormal, v1.EventReasonReconcileComplete),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			toSync := []schema.GroupVersionKind{kinds.Deployment()}
			if !tc.toSyncOverride.Empty() {
				toSync = []schema.GroupVersionKind{tc.toSyncOverride}
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			fakeDecoder := testingfake.NewDecoder(syncertesting.ToUnstructuredList(t, syncertesting.Converter, nil))
			fakeEventRecorder := testingfake.NewEventRecorder(t)
			actual := []runtime.Object{tc.namespaceConfig, tc.namespace}
			actual = append(actual, tc.actual...)
			s := runtime.NewScheme()
			s.AddKnownTypeWithName(kinds.Namespace(), &corev1.Namespace{})
			fakeClient := testingfake.NewClient(t, s, actual...)

			testReconciler := syncerreconcile.NewNamespaceConfigReconciler(ctx,
				client.New(fakeClient, metrics.APICallDuration), fakeClient.Applier(), fakeClient, fakeEventRecorder, fakeDecoder, syncertesting.Now, toSync)

			_, err := testReconciler.Reconcile(
				reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name: tc.namespace.Name,
					},
				})

			fakeClient.Check(t, tc.want...)
			if tc.wantEvent != nil {
				fakeEventRecorder.Check(t, *tc.wantEvent)
			} else {
				fakeEventRecorder.Check(t)
			}
			if err != nil {
				t.Errorf("unexpected reconciliation error: %v", err)
			}
		})
	}
}
