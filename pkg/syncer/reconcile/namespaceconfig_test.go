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
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/syncer/client"
	syncertesting "github.com/google/nomos/pkg/syncer/testing"
	"github.com/google/nomos/pkg/testing/fake"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// event represents a K8S event that was emitted as result of the reconcile.
type event struct {
	// corev1.EventTypeNormal/corev1.EventTypeWarning
	kind   string
	reason string
	// set to true if the event was produced with Eventf (in contrast to Event)
	varargs bool

	obj runtime.Object
}

func deployment(deploymentStrategy appsv1.DeploymentStrategyType, opts ...object.Mutator) *appsv1.Deployment {
	opts = append(opts, func(o *ast.FileObject) {
		o.Object.(*appsv1.Deployment).Spec.Strategy.Type = deploymentStrategy
	}, herrings)
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
	managedNamespace   = namespace(eng, managementEnabled, managedQuotaLabels)
	unmanagedNamespace = namespace(eng)

	namespaceCfg       = namespaceConfig(eng, v1.StateSynced, importToken(token))
	namespaceCfgSynced = namespaceConfig(eng, v1.StateSynced, importToken(token), syncTime(metav1.Time{Time: time.Unix(0, 0)}), syncToken(token))

	managedNamespaceReconcileComplete = &event{
		kind:    corev1.EventTypeNormal,
		reason:  "ReconcileComplete",
		varargs: true,
		obj:     namespaceCfg,
	}
)

func TestManagedNamespaceConfigReconcile(t *testing.T) {
	now = func() metav1.Time {
		return metav1.Time{Time: time.Unix(0, 0)}
	}

	testCases := []struct {
		name               string
		declared           runtime.Object
		actual             runtime.Object
		expectCreate       runtime.Object
		expectUpdate       *diff
		expectDelete       runtime.Object
		expectStatusUpdate *v1.NamespaceConfig
		expectEvent        *event
	}{
		{
			name:               "create from declared state",
			declared:           deployment(appsv1.RollingUpdateDeploymentStrategyType),
			expectCreate:       deployment(appsv1.RollingUpdateDeploymentStrategyType, object.Namespace(eng), managementEnabled, tokenAnnotation),
			expectStatusUpdate: namespaceCfgSynced,
			expectEvent:        managedNamespaceReconcileComplete,
		},
		{
			name:               "do not create if management disabled",
			declared:           deployment(appsv1.RollingUpdateDeploymentStrategyType, managementDisabled),
			expectStatusUpdate: namespaceCfgSynced,
		},
		{
			name:               "do not create if management invalid",
			declared:           deployment(appsv1.RollingUpdateDeploymentStrategyType, managementInvalid),
			expectStatusUpdate: namespaceCfgSynced,
			expectEvent: &event{
				kind:   corev1.EventTypeWarning,
				reason: "InvalidAnnotation",
				obj:    toUnstructured(t, converter, deployment(appsv1.RollingUpdateDeploymentStrategyType, managementInvalid, tokenAnnotation, object.Namespace(eng))),
			},
		},
		{
			name:     "update to declared state",
			declared: deployment(appsv1.RollingUpdateDeploymentStrategyType),
			actual:   deployment(appsv1.RecreateDeploymentStrategyType, managementEnabled),
			expectUpdate: &diff{
				declared: deployment(appsv1.RollingUpdateDeploymentStrategyType, object.Namespace(eng), managementEnabled, tokenAnnotation),
				actual:   deployment(appsv1.RecreateDeploymentStrategyType, managementEnabled),
			},
			expectStatusUpdate: namespaceCfgSynced,
			expectEvent:        managedNamespaceReconcileComplete,
		},
		{
			name:     "update to declared state even if actual managed unset",
			declared: deployment(appsv1.RollingUpdateDeploymentStrategyType),
			actual:   deployment(appsv1.RecreateDeploymentStrategyType),
			expectUpdate: &diff{
				declared: deployment(appsv1.RollingUpdateDeploymentStrategyType, object.Namespace(eng), managementEnabled, tokenAnnotation),
				actual:   deployment(appsv1.RecreateDeploymentStrategyType),
			},
			expectStatusUpdate: namespaceCfgSynced,
			expectEvent:        managedNamespaceReconcileComplete,
		},
		{
			name:     "update to declared state even if actual managed invalid",
			declared: deployment(appsv1.RollingUpdateDeploymentStrategyType),
			actual:   deployment(appsv1.RecreateDeploymentStrategyType, managementInvalid),
			expectUpdate: &diff{
				declared: deployment(appsv1.RollingUpdateDeploymentStrategyType, object.Namespace(eng), managementEnabled, tokenAnnotation),
				actual:   deployment(appsv1.RecreateDeploymentStrategyType, managementInvalid),
			},
			expectStatusUpdate: namespaceCfgSynced,
			expectEvent:        managedNamespaceReconcileComplete,
		},
		{
			name:               "do not update if declared managed invalid",
			declared:           deployment(appsv1.RollingUpdateDeploymentStrategyType, managementInvalid),
			actual:             deployment(appsv1.RecreateDeploymentStrategyType),
			expectStatusUpdate: namespaceCfgSynced,
			expectEvent: &event{
				kind:   corev1.EventTypeWarning,
				reason: "InvalidAnnotation",
				obj:    toUnstructured(t, converter, deployment(appsv1.RollingUpdateDeploymentStrategyType, managementInvalid, tokenAnnotation, object.Namespace(eng))),
			},
		},
		{
			name:     "update to unmanaged",
			declared: deployment(appsv1.RollingUpdateDeploymentStrategyType, managementDisabled),
			actual:   deployment(appsv1.RecreateDeploymentStrategyType, managementEnabled),
			expectUpdate: &diff{
				declared: deployment(appsv1.RecreateDeploymentStrategyType),
				actual:   deployment(appsv1.RecreateDeploymentStrategyType, managementEnabled),
			},
			expectStatusUpdate: namespaceCfgSynced,
			expectEvent:        managedNamespaceReconcileComplete,
		},
		{
			name:               "do not update if unmanaged",
			declared:           deployment(appsv1.RollingUpdateDeploymentStrategyType, managementDisabled),
			actual:             deployment(appsv1.RecreateDeploymentStrategyType),
			expectStatusUpdate: namespaceCfgSynced,
		},
		{
			name:               "delete if managed",
			actual:             deployment(appsv1.RecreateDeploymentStrategyType, managementEnabled),
			expectDelete:       deployment(appsv1.RecreateDeploymentStrategyType, managementEnabled),
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
			actual: deployment(appsv1.RecreateDeploymentStrategyType, managementInvalid),
			expectUpdate: &diff{
				declared: deployment(appsv1.RecreateDeploymentStrategyType),
				actual:   deployment(appsv1.RecreateDeploymentStrategyType, managementInvalid),
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

			tm := newTestMocks(t, mockCtrl)

			fakeDecoder := syncertesting.NewFakeDecoder(toUnstructuredList(t, converter, tc.declared))
			testReconciler := NewNamespaceConfigReconciler(ctx,
				client.New(tm.mockClient), tm.mockApplier, tm.mockCache, tm.mockRecorder, fakeDecoder, toSync)

			tm.expectNamespaceCacheGet(namespaceCfg, managedNamespace)

			tm.expectNamespaceConfigClientGet(namespaceCfg)
			tm.expectNamespaceClientGet(unmanagedNamespace)

			tm.expectNamespaceUpdate(managedNamespace)

			tm.expectCacheList(kinds.Deployment(), managedNamespace.Name, tc.actual)
			tm.expectCreate(tc.expectCreate)
			tm.expectUpdate(tc.expectUpdate)
			tm.expectDelete(tc.expectDelete)

			tm.expectNamespaceStatusUpdate(tc.expectStatusUpdate)
			tm.expectEvent(tc.expectEvent)

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
	now = func() metav1.Time {
		return metav1.Time{Time: time.Unix(0, 0)}
	}

	testCases := []struct {
		name                string
		namespaceConfig     *v1.NamespaceConfig
		namespace           *corev1.Namespace
		wantNamespaceUpdate *corev1.Namespace
		wantStatusUpdate    *v1.NamespaceConfig
		declared            runtime.Object
		actual              runtime.Object
		wantUpdate          *diff
		wantEvent           *event
	}{
		{
			name:                "clean up unmanaged namespace with namespaceconfig",
			namespaceConfig:     namespaceConfig("eng", v1.StateSynced, importToken(token), syncToken(token), managementDisabled),
			namespace:           namespace("eng", managedQuotaLabels, managementEnabled),
			wantNamespaceUpdate: namespace("eng"),
			wantStatusUpdate: namespaceConfig("eng", v1.StateError, importToken(token), syncTime(now()), syncToken(token),
				namespaceSyncError(v1.ConfigManagementError{
					ResourceName: "eng",
					ResourceGVK:  corev1.SchemeGroupVersion.WithKind("Namespace"),
					ErrorMessage: unmanagedError(),
				}), managementDisabled),
			wantEvent: &event{
				kind:   corev1.EventTypeWarning,
				reason: "UnmanagedNamespace",
				obj:    namespace("eng", managedQuotaLabels, managementEnabled),
			},
		},
		{
			name:                "clean up unmanaged namespace without namespaceconfig",
			namespace:           namespace("eng", managedQuotaLabels),
			wantNamespaceUpdate: namespace("eng"),
		},
		{
			name:            "unmanaged namespace has resources synced but status error",
			namespaceConfig: namespaceConfig("eng", v1.StateSynced, importToken(token), managementDisabled),
			namespace:       namespace("eng"),
			declared:        deployment(appsv1.RecreateDeploymentStrategyType),
			actual:          deployment(appsv1.RollingUpdateDeploymentStrategyType, object.Namespace("eng"), tokenAnnotation),
			wantStatusUpdate: namespaceConfig("eng", v1.StateError, importToken(token), syncTime(now()), syncToken(token),
				namespaceSyncError(v1.ConfigManagementError{
					ResourceName: "eng",
					ResourceGVK:  corev1.SchemeGroupVersion.WithKind("Namespace"),
					ErrorMessage: unmanagedError(),
				}), managementDisabled),
			wantEvent: &event{
				kind:   corev1.EventTypeWarning,
				reason: "UnmanagedNamespace",
				obj:    namespace("eng"),
			},
			wantUpdate: &diff{
				declared: deployment(appsv1.RecreateDeploymentStrategyType, object.Namespace("eng"), managementEnabled, tokenAnnotation),
				actual:   deployment(appsv1.RollingUpdateDeploymentStrategyType, object.Namespace("eng"), tokenAnnotation),
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

			tm := newTestMocks(t, mockCtrl)
			fakeDecoder := syncertesting.NewFakeDecoder(toUnstructuredList(t, converter, tc.declared))
			testReconciler := NewNamespaceConfigReconciler(ctx,
				client.New(tm.mockClient), tm.mockApplier, tm.mockCache, tm.mockRecorder, fakeDecoder, toSync)

			tm.expectNamespaceCacheGet(tc.namespaceConfig, tc.namespace)

			tm.expectNamespaceConfigClientGet(tc.namespaceConfig)
			if tc.wantNamespaceUpdate != nil {
				tm.expectNamespaceClientGet(tc.namespace)
			}

			tm.expectUpdate(tc.wantUpdate)
			tm.expectNamespaceUpdate(tc.wantNamespaceUpdate)

			if tc.namespaceConfig != nil {
				tm.expectCacheList(kinds.Deployment(), tc.namespace.Name, tc.actual)
			}

			tm.expectEvent(tc.wantEvent)
			tm.expectNamespaceStatusUpdate(tc.wantStatusUpdate)

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
	now = func() metav1.Time {
		return metav1.Time{Time: time.Unix(0, 0)}
	}
	testCases := []struct {
		name                string
		namespaceConfig     *v1.NamespaceConfig
		namespace           *corev1.Namespace
		wantNamespaceUpdate *corev1.Namespace
		wantStatusUpdate    *v1.NamespaceConfig
	}{
		{
			name:                "do not add quota enforcement label on managed kube-system",
			namespaceConfig:     namespaceConfig("kube-system", v1.StateSynced, importToken(token)),
			namespace:           namespace("kube-system", managementEnabled),
			wantNamespaceUpdate: namespace("kube-system", managementEnabled),
			wantStatusUpdate:    namespaceConfig("kube-system", v1.StateSynced, importToken(token), syncTime(now()), syncToken(token)),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			toSync := []schema.GroupVersionKind{kinds.Deployment()}

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			tm := newTestMocks(t, mockCtrl)
			fakeDecoder := syncertesting.NewFakeDecoder(toUnstructuredList(t, converter, nil))
			testReconciler := NewNamespaceConfigReconciler(ctx,
				client.New(tm.mockClient), tm.mockApplier, tm.mockCache, tm.mockRecorder, fakeDecoder, toSync)

			tm.expectNamespaceCacheGet(tc.namespaceConfig, tc.namespace)

			tm.expectNamespaceConfigClientGet(tc.namespaceConfig)
			tm.expectNamespaceClientGet(unmanaged(tc.namespace))

			tm.expectNamespaceUpdate(tc.wantNamespaceUpdate)

			tm.expectCacheList(kinds.Deployment(), tc.namespace.Name, nil)

			tm.expectNamespaceStatusUpdate(tc.wantStatusUpdate)

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
	now = func() metav1.Time {
		return metav1.Time{Time: time.Unix(0, 0)}
	}
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
		wantEvent *event
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
					v1.SyncTokenAnnotationKey:         "token",
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
					managementEnabled,
					object.Name("my-deployment")),
				deployment(appsv1.RecreateDeploymentStrategyType,
					object.Namespace("default"),
					object.Name("your-deployment")),
			},
			wantDelete: deployment(
				appsv1.RecreateDeploymentStrategyType,
				object.Namespace("default"),
				managementEnabled,
				object.Name("my-deployment")),
			wantEvent: &event{
				kind:    corev1.EventTypeNormal,
				reason:  "ReconcileComplete",
				varargs: true,
				obj:     &v1.NamespaceConfig{},
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
					v1.SyncTokenAnnotationKey:         "token",
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
					managementEnabled,
					object.Name("my-deployment")),
				deployment(appsv1.RecreateDeploymentStrategyType,
					object.Namespace("kube-system"),
					object.Name("your-deployment")),
			},
			wantDelete: deployment(
				appsv1.RecreateDeploymentStrategyType,
				object.Namespace("kube-system"),
				managementEnabled,
				object.Name("my-deployment")),
			wantEvent: &event{
				kind:    corev1.EventTypeNormal,
				reason:  "ReconcileComplete",
				varargs: true,
				obj:     &v1.NamespaceConfig{},
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

			tm := newTestMocks(t, mockCtrl)
			fakeDecoder := syncertesting.NewFakeDecoder(toUnstructuredList(t, converter, nil))
			testReconciler := NewNamespaceConfigReconciler(ctx,
				client.New(tm.mockClient), tm.mockApplier, tm.mockCache, tm.mockRecorder, fakeDecoder, toSync)

			tm.expectNamespaceCacheGet(nil, tc.namespace)
			tm.expectNamespaceClientGet(tc.namespace)

			tm.expectNamespaceUpdate(tc.wantNamespaceUpdate)

			tm.expectCacheList(kinds.Deployment(), tc.namespace.Name, tc.actual...)
			tm.expectDelete(tc.wantDelete)

			tm.expectEvent(tc.wantEvent)

			tm.expectNamespaceStatusUpdate(tc.wantStatusUpdate)

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

func toUnstructured(t *testing.T, converter runtime.UnstructuredConverter, obj runtime.Object) *unstructured.Unstructured {
	if obj == nil {
		return &unstructured.Unstructured{}
	}
	u, err := converter.ToUnstructured(obj)
	if err != nil {
		t.Fatalf("could not convert to unstructured type: %#v", obj)
	}
	return &unstructured.Unstructured{Object: u}
}

func toUnstructuredList(t *testing.T, converter runtime.UnstructuredConverter, objs ...runtime.Object) []*unstructured.Unstructured {
	result := make([]*unstructured.Unstructured, len(objs))
	for i, obj := range objs {
		result[i] = toUnstructured(t, converter, obj)
	}
	return result
}

// cmpDiffMatcher returns true iff cmp.Diff returns empty string.
// Prints the diff if there is one, as the gomock diff is garbage.
type cmpDiffMatcher struct {
	t        *testing.T
	name     string
	expected interface{}
}

// Eq creates a mathcher that compares to expected and prints a diff in case a
// mismatch is found.
func Eq(t *testing.T, expected interface{}) gomock.Matcher {
	return &cmpDiffMatcher{t: t, expected: expected}
}

func EqN(t *testing.T, name string, expected interface{}) gomock.Matcher {
	return &cmpDiffMatcher{t: t, name: name, expected: expected}
}

func (m *cmpDiffMatcher) String() string {
	return fmt.Sprintf("is equal to %v", m.expected)
}

func (m *cmpDiffMatcher) Matches(actual interface{}) bool {
	opt := cmpopts.EquateEmpty() // Disregard empty map vs nil.
	if diff := cmp.Diff(m.expected, actual, opt); diff != "" {
		m.t.Logf("The %q matcher has a diff (expected- +actual):%v\n\n", m.name, diff)
		return false
	}
	return true
}
