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
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/labeling"
	syncertesting "github.com/google/nomos/pkg/syncer/testing"
	"github.com/google/nomos/pkg/testing/object"
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

// event represents a K8S event.
type event struct {
	kind    string
	reason  string
	varargs bool
}

// application contains the arguments needed for Applier's apply calls.
type application struct {
	intended, current runtime.Object
}

func deployment(deploymentStrategy appsv1.DeploymentStrategyType, opts ...object.BuildOpt) *appsv1.Deployment {
	opts = append(opts, func(o *ast.FileObject) {
		o.Object.(*appsv1.Deployment).Spec.Strategy.Type = deploymentStrategy
	})
	return object.Build(kinds.Deployment(), opts...).Object.(*appsv1.Deployment)
}

func managedDeployment(deploymentStrategy appsv1.DeploymentStrategyType, namespace string, opts ...object.BuildOpt) *appsv1.Deployment {
	opts = append(opts,
		object.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
		object.Annotation(v1.SyncTokenAnnotationKey, token),
		object.Namespace(namespace))
	return deployment(deploymentStrategy, opts...)
}

func namespaceConfig(name string, state v1.PolicySyncState, opts ...object.BuildOpt) *v1.NamespaceConfig {
	opts = append(opts, object.Name(name), func(o *ast.FileObject) {
		o.Object.(*v1.NamespaceConfig).Status.SyncState = state
	})
	return object.Build(kinds.NamespaceConfig(), opts...).Object.(*v1.NamespaceConfig)
}

func namespace(name string, opts ...object.BuildOpt) *corev1.Namespace {
	opts = append(opts, object.Name(name))
	return object.Build(kinds.Namespace(), opts...).Object.(*corev1.Namespace)
}

func managedNamespace(name string, opts ...object.BuildOpt) *corev1.Namespace {
	opts = append(opts, object.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled))
	return namespace(name, opts...)
}

func namespaceSyncError(err v1.NamespaceConfigSyncError) object.BuildOpt {
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
		name                string
		namespaceConfig     *v1.NamespaceConfig
		namespace           *corev1.Namespace
		declared            runtime.Object
		actual              runtime.Object
		wantNamespaceUpdate *corev1.Namespace
		wantApply           *application
		wantCreate          runtime.Object
		wantDelete          runtime.Object
		wantStatusUpdate    *v1.NamespaceConfig
		wantEvent           *event
	}{
		{
			name:                "update actual resource to declared state",
			namespaceConfig:     namespaceConfig("eng", v1.StateSynced, importToken(token)),
			namespace:           managedNamespace("eng"),
			declared:            deployment(appsv1.RollingUpdateDeploymentStrategyType),
			actual:              managedDeployment(appsv1.RecreateDeploymentStrategyType, "eng"),
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
			actual:          managedDeployment(appsv1.RollingUpdateDeploymentStrategyType, "eng"),
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
			actual:              deployment(appsv1.RollingUpdateDeploymentStrategyType, object.Namespace("eng"), unmanaged),
			wantNamespaceUpdate: managedNamespace("eng", managedQuotaLabels),
			wantStatusUpdate:    namespaceConfig("eng", v1.StateSynced, importToken(token), syncTime(now()), syncToken(token)),
		},
		{
			name:            "sync unlabeled resource in repo",
			namespaceConfig: namespaceConfig("eng", v1.StateSynced, importToken(token)),
			namespace:       managedNamespace("eng"),
			declared:        deployment(appsv1.RecreateDeploymentStrategyType),
			actual:          deployment(appsv1.RollingUpdateDeploymentStrategyType),
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
			actual: deployment(appsv1.RollingUpdateDeploymentStrategyType,
				object.Annotation(v1.ResourceManagementKey, "invalid")),
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
			actual:              managedDeployment(appsv1.RollingUpdateDeploymentStrategyType, "eng"),
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
			actual:           managedDeployment(appsv1.RollingUpdateDeploymentStrategyType, "eng"),
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
			actual:              deployment(appsv1.RecreateDeploymentStrategyType),
			wantNamespaceUpdate: managedNamespace("eng", managedQuotaLabels),
			wantStatusUpdate:    namespaceConfig("eng", v1.StateSynced, importToken(token), syncTime(now()), syncToken(token)),
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

			var name string
			// Get NamespaceConfig from cache.
			if tc.namespaceConfig == nil {
				name = tc.namespace.GetName()
				mockCache.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.NewNotFound(schema.GroupResource{}, ""))
			} else {
				name = tc.namespaceConfig.GetName()
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
					Get(gomock.Any(), gomock.Any(), gomock.Any())
				mockClient.EXPECT().
					Update(gomock.Any(), Eq(t, ns))
			}

			// List actual resources on the cluster.
			if tc.namespaceConfig != nil {
				mockCache.EXPECT().
					UnstructuredList(gomock.Any(), gomock.Any()).
					Return(toUnstructureds(t, converter, tc.actual), nil)
			}

			// Check for expected creates, applies and deletes.
			if tc.wantCreate != nil {
				mockApplier.EXPECT().
					Create(gomock.Any(), NewUnstructuredMatcher(t, toUnstructured(t, converter, tc.wantCreate)))
			}
			if tc.wantApply != nil {
				mockApplier.EXPECT().
					ApplyNamespace(
						Eq(t, tc.namespaceConfig.Name),
						Eq(t, toUnstructured(t, converter, tc.wantApply.intended)),
						Eq(t, toUnstructured(t, converter, tc.wantApply.current)))
			}
			if tc.wantDelete != nil {
				mockClient.EXPECT().
					Get(gomock.Any(), gomock.Any(), Eq(t, toUnstructured(t, converter, tc.wantDelete)))
				mockClient.EXPECT().
					Delete(gomock.Any(), Eq(t, toUnstructured(t, converter, tc.wantDelete)))
			}

			if tc.wantStatusUpdate != nil {
				// Updates involve first getting the resource from API Server.
				mockClient.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any())
				mockStatusClient := syncertesting.NewMockStatusWriter(mockCtrl)
				mockClient.EXPECT().Status().Return(mockStatusClient)
				mockStatusClient.EXPECT().
					Update(gomock.Any(), Eq(t, tc.wantStatusUpdate))
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
						Name: name,
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

func Eq(t *testing.T, expected interface{}) gomock.Matcher {
	return &cmpDiffMatcher{t: t, expected: expected}
}

func (m *cmpDiffMatcher) String() string {
	return fmt.Sprintf("cmpDiffMatcher Matcher: %v", m.expected)
}

func (m *cmpDiffMatcher) Matches(actual interface{}) bool {
	if diff := cmp.Diff(m.expected, actual); diff != "" {
		m.t.Log(diff)
		return false
	}
	return true
}
