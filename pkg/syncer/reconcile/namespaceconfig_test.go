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
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/labeling"
	syncertesting "github.com/google/nomos/pkg/syncer/testing"
	"github.com/google/nomos/pkg/testing/fake"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/conversion"
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
}

// application contains the arguments needed for Applier's apply calls.
type application struct {
	intended, current runtime.Object
}

func deployment(deploymentStrategy appsv1.DeploymentStrategyType, opts ...object.Mutator) *appsv1.Deployment {
	opts = append(opts, func(o *ast.FileObject) {
		o.Object.(*appsv1.Deployment).Spec.Strategy.Type = deploymentStrategy
	})
	return fake.Build(kinds.Deployment(), opts...).Object.(*appsv1.Deployment)
}

func managedDeployment(deploymentStrategy appsv1.DeploymentStrategyType, namespace string, opts ...object.Mutator) *appsv1.Deployment {
	opts = append(opts,
		object.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
		object.Annotation(v1.SyncTokenAnnotationKey, token),
		object.Namespace(namespace))
	return deployment(deploymentStrategy, opts...)
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

func managedNamespace(name string, opts ...object.Mutator) *corev1.Namespace {
	opts = append(opts, object.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled))
	return namespace(name, opts...)
}

func namespaceSyncError(err v1.NamespaceConfigSyncError) object.Mutator {
	return func(o *ast.FileObject) {
		o.Object.(*v1.NamespaceConfig).Status.SyncErrors = append(o.Object.(*v1.NamespaceConfig).Status.SyncErrors, err)
	}
}

var managedQuotaLabels = object.Labels(labeling.ManageQuota.New())

func TestNamespaceConfigReconcile(t *testing.T) {
	now = func() metav1.Time {
		return metav1.Time{Time: time.Unix(0, 0)}
	}
	testCases := []struct {
		name string
		// What the namespace config looks like for the namespace, set to nil if
		// there is no config.
		namespaceConfig *v1.NamespaceConfig
		// What the namespace resource looks like on the cluster before the
		// reconcile. Set to nil if the namespace is not present.
		namespace *corev1.Namespace
		// The object declared in this namespace config.
		declared runtime.Object
		// The objects present in the corresponding namespace on the cluster.
		actual []runtime.Object
		// If set, forces the setup of expectations for mockCache.
		requireListActual bool
		// What the namespace should look like after an update.
		wantNamespaceUpdate *corev1.Namespace
		// What changes are made to existing objects in the namespace.
		wantApply *application
		// The objects that got created in the namespace.
		wantCreate runtime.Object
		// The objects that got deleted in the namespace.
		wantDelete runtime.Object
		// This is what the NamespaceConfig resource should look like (with
		// updated Status field) on the cluster.
		wantStatusUpdate *v1.NamespaceConfig
		// The events that are expected to be emitted as the result of the
		// operation.
		wantEvent *event
	}{
		{
			name:                "update actual resource to declared state",
			namespaceConfig:     namespaceConfig("eng", v1.StateSynced, importToken(token)),
			namespace:           managedNamespace("eng"),
			declared:            deployment(appsv1.RollingUpdateDeploymentStrategyType),
			actual:              []runtime.Object{managedDeployment(appsv1.RecreateDeploymentStrategyType, "eng")},
			wantNamespaceUpdate: managedNamespace("eng", managedQuotaLabels),
			wantApply: &application{
				intended: managedDeployment(appsv1.RollingUpdateDeploymentStrategyType, "eng"),
				current:  managedDeployment(appsv1.RecreateDeploymentStrategyType, "eng"),
			},
			wantStatusUpdate: namespaceConfig("eng", v1.StateSynced, importToken(token), syncTime(now()), syncToken(token)),
			wantEvent:        reconcileComplete,
		},
		{
			name:            "actual resource already matches declared state",
			namespaceConfig: namespaceConfig("eng", v1.StateSynced, importToken(token), syncToken(token)),
			namespace:       managedNamespace("eng"),
			declared:        deployment(appsv1.RollingUpdateDeploymentStrategyType),
			actual:          []runtime.Object{managedDeployment(appsv1.RollingUpdateDeploymentStrategyType, "eng")},
			wantApply: &application{
				intended: managedDeployment(appsv1.RollingUpdateDeploymentStrategyType, "eng"),
				current:  managedDeployment(appsv1.RollingUpdateDeploymentStrategyType, "eng"),
			},
			wantNamespaceUpdate: managedNamespace("eng", managedQuotaLabels),
			wantEvent:           reconcileComplete,
		},
		{
			name:                "clean up label for unmanaged namespace",
			namespaceConfig:     namespaceConfig("eng", v1.StateSynced, importToken(token), syncToken(token), unmanaged),
			namespace:           namespace("eng", managedQuotaLabels, unmanaged),
			wantStatusUpdate:    namespaceConfig("eng", v1.StateError, importToken(token), syncTime(now()), syncToken(token), namespaceSyncError(v1.NamespaceConfigSyncError{ErrorMessage: unmanagedError()}), unmanaged),
			wantNamespaceUpdate: namespace("eng", unmanaged),
			wantEvent: &event{
				kind:   corev1.EventTypeWarning,
				reason: "UnmanagedNamespace",
			},
		},
		{
			name:                "clean up label for unmanaged namespace without a corresponding namespaceconfig",
			namespace:           namespace("eng", managedQuotaLabels),
			wantNamespaceUpdate: namespace("eng"),
		},
		{
			name:                "un-managed resource cannot be synced",
			namespaceConfig:     namespaceConfig("eng", v1.StateSynced, importToken(token)),
			namespace:           managedNamespace("eng"),
			declared:            deployment(appsv1.RecreateDeploymentStrategyType),
			actual:              []runtime.Object{deployment(appsv1.RollingUpdateDeploymentStrategyType, object.Namespace("eng"), unmanaged)},
			wantNamespaceUpdate: managedNamespace("eng", managedQuotaLabels),
			wantStatusUpdate:    namespaceConfig("eng", v1.StateSynced, importToken(token), syncTime(now()), syncToken(token)),
		},
		{
			name:            "sync unlabeled resource in repo",
			namespaceConfig: namespaceConfig("eng", v1.StateSynced, importToken(token)),
			namespace:       managedNamespace("eng"),
			declared:        deployment(appsv1.RecreateDeploymentStrategyType),
			actual:          []runtime.Object{deployment(appsv1.RollingUpdateDeploymentStrategyType)},
			wantApply: &application{
				intended: managedDeployment(appsv1.RecreateDeploymentStrategyType, "eng"),
				current:  deployment(appsv1.RollingUpdateDeploymentStrategyType),
			},
			wantNamespaceUpdate: managedNamespace("eng", managedQuotaLabels),
			wantStatusUpdate:    namespaceConfig("eng", v1.StateSynced, importToken(token), syncTime(now()), syncToken(token)),
			wantEvent:           reconcileComplete,
		},
		{
			name:            "invalid management label on managed resource",
			namespaceConfig: namespaceConfig("eng", v1.StateSynced, importToken(token)),
			namespace:       managedNamespace("eng", managedQuotaLabels),
			declared:        deployment(appsv1.RecreateDeploymentStrategyType),
			actual: []runtime.Object{deployment(appsv1.RollingUpdateDeploymentStrategyType,
				object.Annotation(v1.ResourceManagementKey, "invalid"))},
			wantNamespaceUpdate: managedNamespace("eng", managedQuotaLabels),
			wantStatusUpdate:    namespaceConfig("eng", v1.StateSynced, importToken(token), syncTime(now()), syncToken(token)),
			wantEvent: &event{
				kind:    corev1.EventTypeWarning,
				reason:  "InvalidAnnotation",
				varargs: true,
			},
		},
		{
			name:                "create resource from declared state",
			namespaceConfig:     namespaceConfig("eng", v1.StateSynced, importToken(token)),
			namespace:           managedNamespace("eng"),
			declared:            deployment(appsv1.RollingUpdateDeploymentStrategyType),
			wantNamespaceUpdate: managedNamespace("eng", managedQuotaLabels),
			wantCreate:          managedDeployment(appsv1.RollingUpdateDeploymentStrategyType, "eng"),
			wantStatusUpdate:    namespaceConfig("eng", v1.StateSynced, importToken(token), syncTime(now()), syncToken(token)),
			wantEvent:           reconcileComplete,
		},
		{
			name:                "delete resource according to declared state",
			namespaceConfig:     namespaceConfig("eng", v1.StateSynced, importToken(token)),
			namespace:           managedNamespace("eng"),
			actual:              []runtime.Object{managedDeployment(appsv1.RollingUpdateDeploymentStrategyType, "eng")},
			wantNamespaceUpdate: managedNamespace("eng", managedQuotaLabels),
			wantDelete:          managedDeployment(appsv1.RollingUpdateDeploymentStrategyType, "eng"),
			wantStatusUpdate:    namespaceConfig("eng", v1.StateSynced, importToken(token), syncTime(now()), syncToken(token)),
			wantEvent:           reconcileComplete,
		},
		{
			name:             "unmanaged namespace has resources synced but status error",
			namespaceConfig:  namespaceConfig("eng", v1.StateSynced, importToken(token)),
			namespace:        namespace("eng", unmanaged),
			declared:         deployment(appsv1.RecreateDeploymentStrategyType),
			actual:           []runtime.Object{managedDeployment(appsv1.RollingUpdateDeploymentStrategyType, "eng")},
			wantStatusUpdate: namespaceConfig("eng", v1.StateError, importToken(token), syncTime(now()), syncToken(token), namespaceSyncError(v1.NamespaceConfigSyncError{ErrorMessage: unmanagedError()})),
			wantEvent: &event{
				kind:   corev1.EventTypeWarning,
				reason: "UnmanagedNamespace",
			},
			wantApply: &application{
				intended: managedDeployment(appsv1.RecreateDeploymentStrategyType, "eng"),
				current:  managedDeployment(appsv1.RollingUpdateDeploymentStrategyType, "eng"),
			},
		},
		{
			name:                "sync namespace with managed unset",
			namespaceConfig:     namespaceConfig("eng", v1.StateSynced, importToken(token)),
			namespace:           namespace("eng"),
			wantNamespaceUpdate: managedNamespace("eng", managedQuotaLabels),
			wantStatusUpdate:    namespaceConfig("eng", v1.StateSynced, importToken(token), syncTime(now()), syncToken(token)),
		},
		{
			name:                "don't delete resource with unset management",
			namespaceConfig:     namespaceConfig("eng", v1.StateSynced, importToken(token)),
			namespace:           managedNamespace("eng"),
			actual:              []runtime.Object{deployment(appsv1.RecreateDeploymentStrategyType)},
			wantNamespaceUpdate: managedNamespace("eng", managedQuotaLabels),
			wantStatusUpdate:    namespaceConfig("eng", v1.StateSynced, importToken(token), syncTime(now()), syncToken(token)),
		},
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
						labeling.ConfigManagementQuotaKey:  "some-quota",
						labeling.ConfigManagementSystemKey: "not-used",
						"some-user-label":                  "some-label-value",
					},
				),
			),
			wantNamespaceUpdate: namespace("default", object.Annotation(
				"some-user-annotation", "some-annotation-value",
			),
				object.Label(
					"some-user-label", "some-label-value",
				),
			),
			requireListActual: true,
			actual: []runtime.Object{
				managedDeployment(
					appsv1.RecreateDeploymentStrategyType, "default",
					object.Name("my-deployment")),
				deployment(appsv1.RecreateDeploymentStrategyType,
					object.Namespace("default"),
					object.Name("your-deployment")),
			},
			wantDelete: managedDeployment(
				appsv1.RecreateDeploymentStrategyType, "default",
				object.Name("my-deployment")),
			wantEvent: reconcileComplete,
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
						labeling.ConfigManagementQuotaKey:  "some-quota",
						labeling.ConfigManagementSystemKey: "not-used",
						"some-user-label":                  "some-label-value",
					},
				),
			),
			wantNamespaceUpdate: namespace("kube-system", object.Annotation(
				"some-user-annotation", "some-annotation-value",
			),
				object.Label(
					"some-user-label", "some-label-value",
				),
			),
			requireListActual: true,
			actual: []runtime.Object{
				managedDeployment(
					appsv1.RecreateDeploymentStrategyType, "kube-system",
					object.Name("my-deployment")),
				deployment(appsv1.RecreateDeploymentStrategyType,
					object.Namespace("kube-system"),
					object.Name("your-deployment")),
			},
			wantDelete: managedDeployment(
				appsv1.RecreateDeploymentStrategyType, "kube-system",
				object.Name("my-deployment")),
			wantEvent: reconcileComplete,
		},
		{
			name:                "do not add quota enforcement label on managed kube-system",
			namespaceConfig:     namespaceConfig("kube-system", v1.StateSynced, importToken(token)),
			namespace:           namespace("kube-system"),
			wantNamespaceUpdate: managedNamespace("kube-system"),
			wantStatusUpdate:    namespaceConfig("kube-system", v1.StateSynced, importToken(token), syncTime(now()), syncToken(token)),
		},
	}

	converter := runtime.NewTestUnstructuredConverter(conversion.EqualitiesOrDie())
	toSync := []schema.GroupVersionKind{kinds.Deployment()}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockClient := syncertesting.NewMockClient(mockCtrl)
			mockApplier := syncertesting.NewMockApplier(mockCtrl)
			mockCache := syncertesting.NewMockGenericCache(mockCtrl)
			mockRecorder := syncertesting.NewMockEventRecorder(mockCtrl)
			fakeDecoder := syncertesting.NewFakeDecoder(toUnstructureds(t, converter, tc.declared))

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			testReconciler := NewNamespaceConfigReconciler(ctx,
				client.New(mockClient), mockApplier, mockCache, mockRecorder, fakeDecoder, toSync)

			// TODO(filmil): Do not use general expectations (one that has all
			// args set to Any() like the one just below.  Because the order of
			// evaluation of multiple EXPECT() calls is not defined, then a
			// more general expectation may mask a less general one, causing
			// the less general one to fail.

			var nsName string
			// Get NamespaceConfig from cache.
			if tc.namespaceConfig == nil {
				nsName = tc.namespace.GetName()
				mockCache.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.NewNotFound(schema.GroupResource{}, ""))
			} else {
				nsName = tc.namespaceConfig.GetName()
				mockCache.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any()).
					SetArg(2, *tc.namespaceConfig)
			}
			// Get Namespace from cache.
			mockCache.EXPECT().
				Get(gomock.Any(), gomock.Any(), gomock.Any()).
				SetArg(2, *tc.namespace)

			// Optionally, update namespace.
			if ns := tc.wantNamespaceUpdate; ns != nil {
				// Updates involve first getting the resource from API Server.
				mockClient.EXPECT().
					Get(gomock.Any(), gomock.Any(), NameAndTypeEq(ns.Name, ns.Namespace, ns))
				mockClient.EXPECT().
					Update(gomock.Any(), Eq(t, ns))
			}

			// List actual resources on the cluster.
			if tc.namespaceConfig != nil || tc.requireListActual {
				mockCache.EXPECT().
					UnstructuredList(gomock.Any(), gomock.Any()).
					Return(toUnstructuredList(t, converter, tc.actual), nil)
			}

			// Check for expected creates, applies and deletes.
			if wc := tc.wantCreate; wc != nil {
				mockApplier.EXPECT().
					Create(gomock.Any(), NewUnstructuredMatcher(t, toUnstructured(t, converter, wc))).
					Return(true, nil)
			}
			if wa := tc.wantApply; wa != nil {
				mockApplier.EXPECT().
					Update(gomock.Any(),
						Eq(t, toUnstructured(t, converter, wa.intended)),
						Eq(t, toUnstructured(t, converter, wa.current))).
					Return(true, nil)
			}
			if wd := tc.wantDelete; wd != nil {
				mockApplier.EXPECT().
					Delete(gomock.Any(),
						Eq(t, toUnstructured(t, converter, wd))).
					Return(true, nil)
			}

			if wsu := tc.wantStatusUpdate; wsu != nil {
				// Updates involve first getting the resource from API Server.
				mockClient.EXPECT().
					Get(gomock.Any(), gomock.Any(),
						NameAndTypeEq(wsu.Name, wsu.Namespace, wsu))
				mockStatusClient := syncertesting.NewMockStatusWriter(mockCtrl)
				mockClient.EXPECT().Status().Return(mockStatusClient)
				mockStatusClient.EXPECT().Update(gomock.Any(), Eq(t, wsu))
			}

			// Check for events with warning or status.
			if tc.wantEvent != nil {
				if tc.wantEvent.varargs {
					mockRecorder.EXPECT().
						Eventf(gomock.Any(), Eq(t, tc.wantEvent.kind), Eq(t, tc.wantEvent.reason), gomock.Any(), gomock.Any())
				} else {
					mockRecorder.EXPECT().
						Event(gomock.Any(), Eq(t, tc.wantEvent.kind), Eq(t, tc.wantEvent.reason), gomock.Any())
				}
			}

			_, err := testReconciler.Reconcile(
				reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name: nsName,
					},
				})
			if err != nil {
				t.Errorf("unexpected reconciliation error: %v", err)
			}
		})
	}
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

func toUnstructureds(t *testing.T, converter runtime.UnstructuredConverter, obj runtime.Object) []*unstructured.Unstructured {
	if obj == nil {
		return nil
	}
	return []*unstructured.Unstructured{toUnstructured(t, converter, obj)}
}

func toUnstructuredList(t *testing.T, converter runtime.UnstructuredConverter,
	objs []runtime.Object) (us []*unstructured.Unstructured) {
	for _, obj := range objs {
		us = append(us, toUnstructured(t, converter, obj))
	}
	return
}

// unstructuredMatcher ignores fields with randomly ordered values in unstructured.Unstructured when comparing.
type unstructuredMatcher struct {
	underlying gomock.Matcher
}

func NewUnstructuredMatcher(t *testing.T, u *unstructured.Unstructured) gomock.Matcher {
	return &unstructuredMatcher{underlying: Eq(t, u)}
}

func (m *unstructuredMatcher) Matches(x interface{}) bool {
	u, ok := x.(*unstructured.Unstructured)
	if !ok {
		return false
	}
	as := u.GetAnnotations()
	delete(as, corev1.LastAppliedConfigAnnotation)
	u.SetAnnotations(as)
	return m.underlying.Matches(x)
}

func (m *unstructuredMatcher) String() string {
	return fmt.Sprintf("unstructured.Unstructured Matcher: %v", m.underlying.String())
}

// cmpDiffMatcher returns true iff cmp.Diff returns empty string.
// Prints the diff if there is one, as the gomock diff is garbage.
type cmpDiffMatcher struct {
	t        *testing.T
	expected interface{}
}

// Eq creates a mathcher that compares to expected and prints a diff in case a
// mismatch is found.
func Eq(t *testing.T, expected interface{}) gomock.Matcher {
	return &cmpDiffMatcher{t: t, expected: expected}
}

func (m *cmpDiffMatcher) String() string {
	return fmt.Sprintf("is equal to %v", m.expected)
}

func (m *cmpDiffMatcher) Matches(actual interface{}) bool {
	opt := cmpopts.EquateEmpty() // Disregard empty map vs nil.
	if diff := cmp.Diff(m.expected, actual, opt); diff != "" {
		m.t.Logf("This matcher has a diff (expected- +actual):%v\n\n", diff)
		return false
	}
	return true
}

// typeNameNamespaceMatcher matches an object by type and name/namespace
type typedNameMatcher struct {
	name, namespace string
	typeMatch       gomock.Matcher
	t               interface{}
}

// NameAndTypeEq matches an object by name, namespace and type. The type must
// be the same go type as the supplied parameter t.
func NameAndTypeEq(name, namespace string, t interface{}) gomock.Matcher {
	return &typedNameMatcher{name: name, namespace: namespace, t: t, typeMatch: gomock.AssignableToTypeOf(t)}
}

func (m *typedNameMatcher) String() string {
	return fmt.Sprintf("is a %T named %q in namespace %q", m.t, m.name, m.namespace)
}

func (m *typedNameMatcher) Matches(x interface{}) bool {
	if !m.typeMatch.Matches(x) {
		return false
	}
	o, ok := x.(metav1.Object)
	if !ok {
		return false
	}
	return o.GetName() == m.name && o.GetNamespace() == m.namespace
}
