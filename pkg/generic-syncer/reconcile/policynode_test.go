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
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/generic-syncer/client"
	syncerdiffer "github.com/google/nomos/pkg/generic-syncer/differ"
	syncertesting "github.com/google/nomos/pkg/generic-syncer/testing"
	"github.com/google/nomos/pkg/syncer/labeling"
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
		wantCreates         []runtime.Object
		wantUpdates         []runtime.Object
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
					ImportToken: "abc123",
				},
				Status: v1.PolicyNodeStatus{
					SyncState: v1.StateSynced,
				},
			},
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "eng",
					Labels: labeling.ManageAll.New(),
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
				ObjectMeta: metav1.ObjectMeta{
					Name: "eng",
					Labels: map[string]string{
						labeling.ManagementKey: labeling.Full,
					},
				},
			},
			wantUpdates: []runtime.Object{
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
							v1alpha1.SyncTokenAnnotationKey: "abc123",
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
					ImportToken: "abc123",
				},
				Status: v1.PolicyNodeStatus{
					SyncState: v1.StateSynced,
					SyncTime:  now(),
					SyncToken: "abc123",
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
					ImportToken: "abc123",
				},
				Status: v1.PolicyNodeStatus{
					SyncState: v1.StateSynced,
					SyncToken: "abc123",
				},
			},
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "eng",
					Labels: labeling.ManageAll.New(),
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
							v1alpha1.SyncTokenAnnotationKey: "abc123",
						},
					},
					Spec: appsv1.DeploymentSpec{
						Strategy: appsv1.DeploymentStrategy{
							Type: appsv1.RollingUpdateDeploymentStrategyType,
						},
					},
				},
			},
			wantNamespaceUpdate: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "eng",
					Labels: map[string]string{
						labeling.ManagementKey: labeling.Full,
					},
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
					ImportToken: "abc123",
				},
				Status: v1.PolicyNodeStatus{
					SyncState: v1.StateSynced,
				},
			},
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "eng",
					Labels: labeling.ManageAll.New(),
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
				ObjectMeta: metav1.ObjectMeta{
					Name: "eng",
					Labels: map[string]string{
						labeling.ManagementKey: labeling.Full,
					},
				},
			},
			wantStatusUpdate: &v1.PolicyNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "eng",
				},
				Spec: v1.PolicyNodeSpec{
					Type:        v1.Namespace,
					ImportToken: "abc123",
				},
				Status: v1.PolicyNodeStatus{
					SyncState: v1.StateSynced,
					SyncTime:  now(),
					SyncToken: "abc123",
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
					ImportToken: "abc123",
				},
				Status: v1.PolicyNodeStatus{
					SyncState: v1.StateSynced,
				},
			},
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "eng",
					Labels: labeling.ManageAll.New(),
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
				ObjectMeta: metav1.ObjectMeta{
					Name: "eng",
					Labels: map[string]string{
						labeling.ManagementKey: labeling.Full,
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
							v1alpha1.SyncTokenAnnotationKey: "abc123",
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
					ImportToken: "abc123",
				},
				Status: v1.PolicyNodeStatus{
					SyncState: v1.StateSynced,
					SyncTime:  now(),
					SyncToken: "abc123",
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
					ImportToken: "abc123",
				},
				Status: v1.PolicyNodeStatus{
					SyncState: v1.StateSynced,
				},
			},
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "eng",
					Labels: labeling.ManageAll.New(),
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
				ObjectMeta: metav1.ObjectMeta{
					Name: "eng",
					Labels: map[string]string{
						labeling.ManagementKey: labeling.Full,
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
					ImportToken: "abc123",
				},
				Status: v1.PolicyNodeStatus{
					SyncState: v1.StateSynced,
					SyncTime:  now(),
					SyncToken: "abc123",
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
			name: "un-managed namespace cannot have its resources synced",
			policyNode: &v1.PolicyNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "eng",
				},
				Spec: v1.PolicyNodeSpec{
					Type:        v1.Namespace,
					ImportToken: "abc123",
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
			wantEvents: []event{
				{
					kind:   corev1.EventTypeWarning,
					reason: "UnmanagedNamespace",
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
	comparator := syncerdiffer.NewComparator([]*v1alpha1.Sync{sync(gvk)}, labeling.ResourceManagementKey)
	toSync := []schema.GroupVersionKind{gvk}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockClient := syncertesting.NewMockClient(mockCtrl)
			mockCache := syncertesting.NewMockGenericCache(mockCtrl)
			mockRecorder := syncertesting.NewMockEventRecorder(mockCtrl)
			fakeDecoder := syncertesting.NewFakeDecoder(toUnstructureds(t, converter, tc.declared))

			testReconciler := NewPolicyNodeReconciler(
				client.New(mockClient), mockCache, mockRecorder, fakeDecoder, comparator, toSync)

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
					UnstructuredList(gomock.Any()).
					Return(toUnstructureds(t, converter, tc.actual), nil)
			}

			// Check for expected create, update and deletes.
			for _, wantCreate := range tc.wantCreates {
				mockClient.EXPECT().
					Create(gomock.Any(), gomock.Eq(toUnstructured(t, converter, wantCreate)))
			}
			for _, wantUpdate := range tc.wantUpdates {
				// Updates involve first getting the resource from API Server.
				mockClient.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any())
				mockClient.EXPECT().
					Update(gomock.Any(), gomock.Eq(toUnstructured(t, converter, wantUpdate)))
			}
			for _, wantDelete := range tc.wantDeletes {
				mockClient.EXPECT().
					Delete(gomock.Any(), gomock.Eq(toUnstructured(t, converter, wantDelete)))
			}

			if tc.wantStatusUpdate != nil {
				// Updates involve first getting the resource from API Server.
				mockClient.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any())
				mockClient.EXPECT().
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

func sync(gvk schema.GroupVersionKind) *v1alpha1.Sync {
	return &v1alpha1.Sync{
		ObjectMeta: metav1.ObjectMeta{
			Finalizers: []string{},
		},
		Spec: v1alpha1.SyncSpec{
			Groups: []v1alpha1.SyncGroup{
				{
					Group: gvk.Group,
					Kinds: []v1alpha1.SyncKind{
						{
							Kind: gvk.Kind,
							Versions: []v1alpha1.SyncVersion{
								{
									Version: gvk.Version,
								},
							},
						},
					},
				},
			},
		},
	}
}
