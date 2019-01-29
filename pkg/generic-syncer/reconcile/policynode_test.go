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
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/generic-syncer/client"
	"github.com/google/nomos/pkg/generic-syncer/labeling"
	syncertesting "github.com/google/nomos/pkg/generic-syncer/testing"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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

func TestPolicyNodeReconcile(t *testing.T) {
	now = func() metav1.Time {
		return metav1.Time{Time: time.Unix(0, 0)}
	}
	testCases := []struct {
		name                string
		policyNode          *v1.PolicyNode
		namespace           *corev1.Namespace
		declared            []runtime.Object
		actual              []runtime.Object
		wantNamespaceUpdate *corev1.Namespace
		wantApplies         []application
		wantCreates         []runtime.Object
		wantDeletes         []runtime.Object
		wantStatusUpdate    *v1.PolicyNode
		wantEvents          []event
	}{
		{
			name: "update actual resource to declared state",
			policyNode: &v1.PolicyNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "eng",
				},
				Spec: v1.PolicyNodeSpec{
					Type:        v1.Namespace,
					ImportToken: "b38239ea8f58eaed17af6734bd6a025eeafccda1",
				},
				Status: v1.PolicyNodeStatus{
					SyncState: v1.StateSynced,
				},
			},
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "eng",
					Labels: labeling.ManageResource.New(),
				},
			},
			declared: []runtime.Object{
				&appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-deployment",
					},
					Spec: appsv1.DeploymentSpec{
						Strategy: appsv1.DeploymentStrategy{
							Type: appsv1.RollingUpdateDeploymentStrategyType,
						},
					},
				},
			},
			actual: []runtime.Object{
				&appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-deployment",
						Namespace: "eng",
						Labels:    labeling.ManageResource.New(),
					},
					Spec: appsv1.DeploymentSpec{
						Strategy: appsv1.DeploymentStrategy{
							Type: appsv1.RecreateDeploymentStrategyType,
						},
					},
				},
			},
			wantNamespaceUpdate: &corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "eng",
					Labels: map[string]string{
						labeling.ResourceManagementKey: labeling.Enabled,
					},
				},
			},
			wantApplies: []application{
				{
					intended: &appsv1.Deployment{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Deployment",
							APIVersion: "apps/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-deployment",
							Namespace: "eng",
							Labels:    labeling.ManageResource.New(),
							Annotations: map[string]string{
								v1alpha1.SyncTokenAnnotationKey: "b38239ea8f58eaed17af6734bd6a025eeafccda1",
							},
						},
						Spec: appsv1.DeploymentSpec{
							Strategy: appsv1.DeploymentStrategy{
								Type: appsv1.RollingUpdateDeploymentStrategyType,
							},
						},
					},
					current: &appsv1.Deployment{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Deployment",
							APIVersion: "apps/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-deployment",
							Namespace: "eng",
							Labels:    labeling.ManageResource.New(),
						},
						Spec: appsv1.DeploymentSpec{
							Strategy: appsv1.DeploymentStrategy{
								Type: appsv1.RecreateDeploymentStrategyType,
							},
						},
					},
				},
			},
			wantStatusUpdate: &v1.PolicyNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "eng",
				},
				Spec: v1.PolicyNodeSpec{
					Type:        v1.Namespace,
					ImportToken: "b38239ea8f58eaed17af6734bd6a025eeafccda1",
				},
				Status: v1.PolicyNodeStatus{
					SyncState: v1.StateSynced,
					SyncTime:  now(),
					SyncToken: "b38239ea8f58eaed17af6734bd6a025eeafccda1",
				},
			},
			wantEvents: []event{
				{
					kind:    corev1.EventTypeNormal,
					reason:  "ReconcileComplete",
					varargs: true,
				},
			},
		},
		{
			name: "actual resource already matches declared state",
			policyNode: &v1.PolicyNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "eng",
				},
				Spec: v1.PolicyNodeSpec{
					Type:        v1.Namespace,
					ImportToken: "b38239ea8f58eaed17af6734bd6a025eeafccda1",
				},
				Status: v1.PolicyNodeStatus{
					SyncState: v1.StateSynced,
					SyncToken: "b38239ea8f58eaed17af6734bd6a025eeafccda1",
				},
			},
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "eng",
					Labels: labeling.ManageResource.New(),
				},
			},
			declared: []runtime.Object{
				&appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-deployment",
					},
					Spec: appsv1.DeploymentSpec{
						Strategy: appsv1.DeploymentStrategy{
							Type: appsv1.RollingUpdateDeploymentStrategyType,
						},
					},
				},
			},
			actual: []runtime.Object{
				&appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-deployment",
						Namespace: "eng",
						Labels:    labeling.ManageResource.New(),
						Annotations: map[string]string{
							v1alpha1.SyncTokenAnnotationKey: "b38239ea8f58eaed17af6734bd6a025eeafccda1",
						},
					},
					Spec: appsv1.DeploymentSpec{
						Strategy: appsv1.DeploymentStrategy{
							Type: appsv1.RollingUpdateDeploymentStrategyType,
						},
					},
				},
			},
			wantApplies: []application{
				{
					intended: &appsv1.Deployment{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Deployment",
							APIVersion: "apps/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-deployment",
							Namespace: "eng",
							Labels:    labeling.ManageResource.New(),
							Annotations: map[string]string{
								v1alpha1.SyncTokenAnnotationKey: "b38239ea8f58eaed17af6734bd6a025eeafccda1",
							},
						},
						Spec: appsv1.DeploymentSpec{
							Strategy: appsv1.DeploymentStrategy{
								Type: appsv1.RollingUpdateDeploymentStrategyType,
							},
						},
					},
					current: &appsv1.Deployment{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Deployment",
							APIVersion: "apps/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-deployment",
							Namespace: "eng",
							Labels:    labeling.ManageResource.New(),
							Annotations: map[string]string{
								v1alpha1.SyncTokenAnnotationKey: "b38239ea8f58eaed17af6734bd6a025eeafccda1",
							},
						},
						Spec: appsv1.DeploymentSpec{
							Strategy: appsv1.DeploymentStrategy{
								Type: appsv1.RollingUpdateDeploymentStrategyType,
							},
						},
					},
				},
			},
			wantNamespaceUpdate: &corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "eng",
					Labels: map[string]string{
						labeling.ResourceManagementKey: labeling.Enabled,
					},
				},
			},
			wantEvents: []event{
				{
					kind:    corev1.EventTypeNormal,
					reason:  "ReconcileComplete",
					varargs: true,
				},
			},
		},
		{
			name: "un-managed resource cannot be synced",
			policyNode: &v1.PolicyNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "eng",
				},
				Spec: v1.PolicyNodeSpec{
					Type:        v1.Namespace,
					ImportToken: "b38239ea8f58eaed17af6734bd6a025eeafccda1",
				},
				Status: v1.PolicyNodeStatus{
					SyncState: v1.StateSynced,
				},
			},
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "eng",
					Labels: labeling.ManageResource.New(),
				},
			},
			declared: []runtime.Object{
				&appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-deployment",
					},
					Spec: appsv1.DeploymentSpec{
						Strategy: appsv1.DeploymentStrategy{
							Type: appsv1.RecreateDeploymentStrategyType,
						},
					},
				},
			},
			actual: []runtime.Object{
				&appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-deployment",
						Namespace: "eng",
					},
					Spec: appsv1.DeploymentSpec{
						Strategy: appsv1.DeploymentStrategy{
							Type: appsv1.RollingUpdateDeploymentStrategyType,
						},
					},
				},
			},
			wantNamespaceUpdate: &corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "eng",
					Labels: map[string]string{
						labeling.ResourceManagementKey: labeling.Enabled,
					},
				},
			},
			wantStatusUpdate: &v1.PolicyNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "eng",
				},
				Spec: v1.PolicyNodeSpec{
					Type:        v1.Namespace,
					ImportToken: "b38239ea8f58eaed17af6734bd6a025eeafccda1",
				},
				Status: v1.PolicyNodeStatus{
					SyncState: v1.StateSynced,
					SyncTime:  now(),
					SyncToken: "b38239ea8f58eaed17af6734bd6a025eeafccda1",
				},
			},
			wantEvents: []event{
				{
					kind:    corev1.EventTypeWarning,
					reason:  "UnmanagedResource",
					varargs: true,
				},
			},
		},
		{
			name: "create resource from declared state",
			policyNode: &v1.PolicyNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "eng",
				},
				Spec: v1.PolicyNodeSpec{
					Type:        v1.Namespace,
					ImportToken: "b38239ea8f58eaed17af6734bd6a025eeafccda1",
				},
				Status: v1.PolicyNodeStatus{
					SyncState: v1.StateSynced,
				},
			},
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "eng",
					Labels: labeling.ManageResource.New(),
				},
			},
			declared: []runtime.Object{
				&appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-deployment",
					},
					Spec: appsv1.DeploymentSpec{
						Strategy: appsv1.DeploymentStrategy{
							Type: appsv1.RollingUpdateDeploymentStrategyType,
						},
					},
				},
			},
			actual: []runtime.Object{},
			wantNamespaceUpdate: &corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "eng",
					Labels: map[string]string{
						labeling.ResourceManagementKey: labeling.Enabled,
					},
				},
			},
			wantCreates: []runtime.Object{
				&appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-deployment",
						Namespace: "eng",
						Labels:    labeling.ManageResource.New(),
						Annotations: map[string]string{
							v1alpha1.SyncTokenAnnotationKey: "b38239ea8f58eaed17af6734bd6a025eeafccda1",
						},
					},
					Spec: appsv1.DeploymentSpec{
						Strategy: appsv1.DeploymentStrategy{
							Type: appsv1.RollingUpdateDeploymentStrategyType,
						},
					},
				},
			},
			wantStatusUpdate: &v1.PolicyNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "eng",
				},
				Spec: v1.PolicyNodeSpec{
					Type:        v1.Namespace,
					ImportToken: "b38239ea8f58eaed17af6734bd6a025eeafccda1",
				},
				Status: v1.PolicyNodeStatus{
					SyncState: v1.StateSynced,
					SyncTime:  now(),
					SyncToken: "b38239ea8f58eaed17af6734bd6a025eeafccda1",
				},
			},
			wantEvents: []event{
				{
					kind:    corev1.EventTypeNormal,
					reason:  "ReconcileComplete",
					varargs: true,
				},
			},
		},
		{
			name: "delete resource according to declared state",
			policyNode: &v1.PolicyNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "eng",
				},
				Spec: v1.PolicyNodeSpec{
					Type:        v1.Namespace,
					ImportToken: "b38239ea8f58eaed17af6734bd6a025eeafccda1",
				},
				Status: v1.PolicyNodeStatus{
					SyncState: v1.StateSynced,
				},
			},
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "eng",
					Labels: labeling.ManageResource.New(),
				},
			},
			declared: []runtime.Object{},
			actual: []runtime.Object{
				&appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-deployment",
						Namespace: "eng",
						Labels:    labeling.ManageResource.New(),
					},
					Spec: appsv1.DeploymentSpec{
						Strategy: appsv1.DeploymentStrategy{
							Type: appsv1.RollingUpdateDeploymentStrategyType,
						},
					},
				},
			},
			wantNamespaceUpdate: &corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "eng",
					Labels: map[string]string{
						labeling.ResourceManagementKey: labeling.Enabled,
					},
				},
			},
			wantDeletes: []runtime.Object{
				&appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-deployment",
						Namespace: "eng",
						Labels:    labeling.ManageResource.New(),
					},
					Spec: appsv1.DeploymentSpec{
						Strategy: appsv1.DeploymentStrategy{
							Type: appsv1.RollingUpdateDeploymentStrategyType,
						},
					},
				},
			},
			wantStatusUpdate: &v1.PolicyNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "eng",
				},
				Spec: v1.PolicyNodeSpec{
					Type:        v1.Namespace,
					ImportToken: "b38239ea8f58eaed17af6734bd6a025eeafccda1",
				},
				Status: v1.PolicyNodeStatus{
					SyncState: v1.StateSynced,
					SyncTime:  now(),
					SyncToken: "b38239ea8f58eaed17af6734bd6a025eeafccda1",
				},
			},
			wantEvents: []event{
				{
					kind:    corev1.EventTypeNormal,
					reason:  "ReconcileComplete",
					varargs: true,
				},
			},
		},
		{
			name: "unmanaged namespace has resources synced",
			policyNode: &v1.PolicyNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "eng",
				},
				Spec: v1.PolicyNodeSpec{
					Type:        v1.Namespace,
					ImportToken: "b38239ea8f58eaed17af6734bd6a025eeafccda1",
				},
				Status: v1.PolicyNodeStatus{
					SyncState: v1.StateSynced,
				},
			},
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "eng",
				},
			},
			declared: []runtime.Object{
				&appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-deployment",
					},
					Spec: appsv1.DeploymentSpec{
						Strategy: appsv1.DeploymentStrategy{
							Type: appsv1.RecreateDeploymentStrategyType,
						},
					},
				},
			},
			actual: []runtime.Object{
				&appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-deployment",
						Namespace: "eng",
						Labels:    labeling.ManageResource.New(),
					},
					Spec: appsv1.DeploymentSpec{
						Strategy: appsv1.DeploymentStrategy{
							Type: appsv1.RollingUpdateDeploymentStrategyType,
						},
					},
				},
			},
			wantStatusUpdate: &v1.PolicyNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "eng",
				},
				Spec: v1.PolicyNodeSpec{
					Type:        v1.Namespace,
					ImportToken: "b38239ea8f58eaed17af6734bd6a025eeafccda1",
				},
				Status: v1.PolicyNodeStatus{
					SyncState: v1.StateError,
					SyncTime:  now(),
					SyncToken: "b38239ea8f58eaed17af6734bd6a025eeafccda1",
					SyncErrors: []v1.PolicyNodeSyncError{
						{
							ErrorMessage: fmt.Sprintf("Namespace is missing proper management label (%s=%s)",
								labeling.ResourceManagementKey, labeling.Enabled),
						},
					},
				},
			},
			wantEvents: []event{
				{
					kind:   corev1.EventTypeWarning,
					reason: "UnmanagedNamespace",
				},
			},
			wantApplies: []application{
				{
					intended: &appsv1.Deployment{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Deployment",
							APIVersion: "apps/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-deployment",
							Namespace: "eng",
							Labels:    labeling.ManageResource.New(),
							Annotations: map[string]string{
								v1alpha1.SyncTokenAnnotationKey: "b38239ea8f58eaed17af6734bd6a025eeafccda1",
							},
						},
						Spec: appsv1.DeploymentSpec{
							Strategy: appsv1.DeploymentStrategy{
								Type: appsv1.RecreateDeploymentStrategyType,
							},
						},
					},
					current: &appsv1.Deployment{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Deployment",
							APIVersion: "apps/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-deployment",
							Namespace: "eng",
							Labels:    labeling.ManageResource.New(),
						},
						Spec: appsv1.DeploymentSpec{
							Strategy: appsv1.DeploymentStrategy{
								Type: appsv1.RollingUpdateDeploymentStrategyType,
							},
						},
					},
				},
			},
		},
	}

	converter := runtime.NewTestUnstructuredConverter(conversion.EqualitiesOrDie())
	gvk := schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	}
	toSync := []schema.GroupVersionKind{gvk}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockClient := syncertesting.NewMockClient(mockCtrl)
			mockApplier := syncertesting.NewMockApplier(mockCtrl)
			mockCache := syncertesting.NewMockGenericCache(mockCtrl)
			mockRecorder := syncertesting.NewMockEventRecorder(mockCtrl)
			fakeDecoder := syncertesting.NewFakeDecoder(toUnstructureds(t, converter, tc.declared))

			testReconciler := NewPolicyNodeReconciler(
				client.New(mockClient), mockApplier, mockCache, mockRecorder, fakeDecoder, toSync)

			// Get PolicyNode from cache.
			mockCache.EXPECT().
				Get(gomock.Any(), gomock.Any(), gomock.Any()).
				SetArg(2, *tc.policyNode)
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
					Update(gomock.Any(), gomock.Eq(ns))
			}

			// List actual resources on the cluster.
			if tc.actual != nil {
				mockCache.EXPECT().
					UnstructuredList(gomock.Any(), gomock.Any()).
					Return(toUnstructureds(t, converter, tc.actual), nil)
			}

			// Check for expected creates, applies and deletes.
			for _, wantCreate := range tc.wantCreates {
				mockApplier.EXPECT().
					Create(gomock.Any(), NewUnstructuredMatcher(toUnstructured(t, converter, wantCreate)))
			}
			for _, wantApply := range tc.wantApplies {
				mockApplier.EXPECT().
					ApplyNamespace(
						gomock.Eq(tc.policyNode.Name),
						gomock.Eq(toUnstructured(t, converter, wantApply.intended)),
						gomock.Eq(toUnstructured(t, converter, wantApply.current)))
			}
			for _, wantDelete := range tc.wantDeletes {
				mockClient.EXPECT().
					Delete(gomock.Any(), gomock.Eq(toUnstructured(t, converter, wantDelete)))
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
				if wantEvent.varargs {
					mockRecorder.EXPECT().
						Eventf(gomock.Any(), gomock.Eq(wantEvent.kind), gomock.Eq(wantEvent.reason), gomock.Any(), gomock.Any())
				} else {
					mockRecorder.EXPECT().
						Event(gomock.Any(), gomock.Eq(wantEvent.kind), gomock.Eq(wantEvent.reason), gomock.Any())
				}
			}

			_, err := testReconciler.Reconcile(
				reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name: tc.policyNode.Name,
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

func toUnstructureds(t *testing.T, converter runtime.UnstructuredConverter,
	objs []runtime.Object) (us []*unstructured.Unstructured) {
	for _, obj := range objs {
		us = append(us, toUnstructured(t, converter, obj))
	}
	return
}

// unstructuredMatcher ignores fields with randomly ordered values in unstructured.Unstructured when comparing.
type unstructuredMatcher struct {
	u *unstructured.Unstructured
}

func NewUnstructuredMatcher(u *unstructured.Unstructured) gomock.Matcher {
	return &unstructuredMatcher{u: u}
}

func (m *unstructuredMatcher) Matches(x interface{}) bool {
	u, ok := x.(*unstructured.Unstructured)
	if !ok {
		return false
	}
	as := u.GetAnnotations()
	delete(as, corev1.LastAppliedConfigAnnotation)
	u.SetAnnotations(as)
	return reflect.DeepEqual(u, m.u)
}

func (m *unstructuredMatcher) String() string {
	return fmt.Sprintf("unstructured.Unstructured Matcher: %v", m.u)
}
