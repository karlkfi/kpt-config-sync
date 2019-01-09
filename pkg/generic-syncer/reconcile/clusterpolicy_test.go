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
	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/generic-syncer/client"
	syncerdiffer "github.com/google/nomos/pkg/generic-syncer/differ"
	"github.com/google/nomos/pkg/generic-syncer/labeling"
	syncertesting "github.com/google/nomos/pkg/generic-syncer/testing"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestClusterPolicyReconcile(t *testing.T) {
	now = func() metav1.Time {
		return metav1.Time{Time: time.Unix(0, 0)}
	}
	testCases := []struct {
		name             string
		clusterPolicy    *v1.ClusterPolicy
		declared         []runtime.Object
		actual           []runtime.Object
		wantApplies      []application
		wantCreates      []runtime.Object
		wantDeletes      []runtime.Object
		wantStatusUpdate *v1.ClusterPolicy
		wantEvents       []event
	}{
		{
			name: "update actual resource to declared state",
			clusterPolicy: &v1.ClusterPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: v1.ClusterPolicyName,
				},
				Spec: v1.ClusterPolicySpec{
					ImportToken: "b38239ea8f58eaed17af6734bd6a025eeafccda1",
				},
				Status: v1.ClusterPolicyStatus{
					SyncState: v1.StateSynced,
				},
			},
			declared: []runtime.Object{
				&corev1.PersistentVolume{
					TypeMeta: metav1.TypeMeta{
						Kind:       "PersistentVolume",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-persistentvolume",
					},
					Spec: corev1.PersistentVolumeSpec{
						PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRecycle,
					},
				},
			},
			actual: []runtime.Object{
				&corev1.PersistentVolume{
					TypeMeta: metav1.TypeMeta{
						Kind:       "PersistentVolume",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:   "my-persistentvolume",
						Labels: labeling.ManageResource.New(),
					},
					Spec: corev1.PersistentVolumeSpec{
						PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
					},
				},
			},
			wantApplies: []application{
				{
					intended: &corev1.PersistentVolume{
						TypeMeta: metav1.TypeMeta{
							Kind:       "PersistentVolume",
							APIVersion: "v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:   "my-persistentvolume",
							Labels: labeling.ManageResource.New(),
							Annotations: map[string]string{
								v1alpha1.SyncTokenAnnotationKey: "b38239ea8f58eaed17af6734bd6a025eeafccda1",
							},
						},
						Spec: corev1.PersistentVolumeSpec{
							PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRecycle,
						},
					},
					current: &corev1.PersistentVolume{
						TypeMeta: metav1.TypeMeta{
							Kind:       "PersistentVolume",
							APIVersion: "v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:   "my-persistentvolume",
							Labels: labeling.ManageResource.New(),
						},
						Spec: corev1.PersistentVolumeSpec{
							PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
						},
					},
				},
			},
			wantStatusUpdate: &v1.ClusterPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: v1.ClusterPolicyName,
				},
				Spec: v1.ClusterPolicySpec{
					ImportToken: "b38239ea8f58eaed17af6734bd6a025eeafccda1",
				},
				Status: v1.ClusterPolicyStatus{
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
			clusterPolicy: &v1.ClusterPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: v1.ClusterPolicyName,
				},
				Spec: v1.ClusterPolicySpec{
					ImportToken: "b38239ea8f58eaed17af6734bd6a025eeafccda1",
				},
				Status: v1.ClusterPolicyStatus{
					SyncState: v1.StateSynced,
				},
			},
			declared: []runtime.Object{
				&corev1.PersistentVolume{
					TypeMeta: metav1.TypeMeta{
						Kind:       "PersistentVolume",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-persistentvolume",
					},
					Spec: corev1.PersistentVolumeSpec{
						PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRecycle,
					},
				},
			},
			actual: []runtime.Object{
				&corev1.PersistentVolume{
					TypeMeta: metav1.TypeMeta{
						Kind:       "PersistentVolume",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:   "my-persistentvolume",
						Labels: labeling.ManageResource.New(),
						Annotations: map[string]string{
							v1alpha1.SyncTokenAnnotationKey: "b38239ea8f58eaed17af6734bd6a025eeafccda1",
						},
					},
					Spec: corev1.PersistentVolumeSpec{
						PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRecycle,
					},
				},
			},
			wantApplies: []application{
				{
					intended: &corev1.PersistentVolume{
						TypeMeta: metav1.TypeMeta{
							Kind:       "PersistentVolume",
							APIVersion: "v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:   "my-persistentvolume",
							Labels: labeling.ManageResource.New(),
							Annotations: map[string]string{
								v1alpha1.SyncTokenAnnotationKey: "b38239ea8f58eaed17af6734bd6a025eeafccda1",
							},
						},
						Spec: corev1.PersistentVolumeSpec{
							PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRecycle,
						},
					},
					current: &corev1.PersistentVolume{
						TypeMeta: metav1.TypeMeta{
							Kind:       "PersistentVolume",
							APIVersion: "v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:   "my-persistentvolume",
							Labels: labeling.ManageResource.New(),
							Annotations: map[string]string{
								v1alpha1.SyncTokenAnnotationKey: "b38239ea8f58eaed17af6734bd6a025eeafccda1",
							},
						},
						Spec: corev1.PersistentVolumeSpec{
							PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRecycle,
						},
					},
				},
			},
			wantStatusUpdate: &v1.ClusterPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: v1.ClusterPolicyName,
				},
				Spec: v1.ClusterPolicySpec{
					ImportToken: "b38239ea8f58eaed17af6734bd6a025eeafccda1",
				},
				Status: v1.ClusterPolicyStatus{
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
			name: "un-managed resource cannot be synced",
			clusterPolicy: &v1.ClusterPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: v1.ClusterPolicyName,
				},
				Status: v1.ClusterPolicyStatus{
					SyncState: v1.StateSynced,
				},
			},
			declared: []runtime.Object{
				&corev1.PersistentVolume{
					TypeMeta: metav1.TypeMeta{
						Kind:       "PersistentVolume",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-persistentvolume",
					},
					Spec: corev1.PersistentVolumeSpec{
						PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRecycle,
					},
				},
			},
			actual: []runtime.Object{
				&corev1.PersistentVolume{
					TypeMeta: metav1.TypeMeta{
						Kind:       "PersistentVolume",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-persistentvolume",
					},
					Spec: corev1.PersistentVolumeSpec{
						PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
					},
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
			clusterPolicy: &v1.ClusterPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: v1.ClusterPolicyName,
				},
				Spec: v1.ClusterPolicySpec{
					ImportToken: "b38239ea8f58eaed17af6734bd6a025eeafccda1",
				},
				Status: v1.ClusterPolicyStatus{
					SyncState: v1.StateSynced,
				},
			},
			declared: []runtime.Object{
				&corev1.PersistentVolume{
					TypeMeta: metav1.TypeMeta{
						Kind:       "PersistentVolume",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-persistentvolume",
					},
					Spec: corev1.PersistentVolumeSpec{
						PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRecycle,
					},
				},
			},
			actual: []runtime.Object{},
			wantCreates: []runtime.Object{
				&corev1.PersistentVolume{
					TypeMeta: metav1.TypeMeta{
						Kind:       "PersistentVolume",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:   "my-persistentvolume",
						Labels: labeling.ManageResource.New(),
						Annotations: map[string]string{
							v1alpha1.SyncTokenAnnotationKey: "b38239ea8f58eaed17af6734bd6a025eeafccda1",
						},
					},
					Spec: corev1.PersistentVolumeSpec{
						PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRecycle,
					},
				},
			},
			wantStatusUpdate: &v1.ClusterPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: v1.ClusterPolicyName,
				},
				Spec: v1.ClusterPolicySpec{
					ImportToken: "b38239ea8f58eaed17af6734bd6a025eeafccda1",
				},
				Status: v1.ClusterPolicyStatus{
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
			clusterPolicy: &v1.ClusterPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: v1.ClusterPolicyName,
				},
				Status: v1.ClusterPolicyStatus{
					SyncState: v1.StateSynced,
				},
			},
			declared: []runtime.Object{},
			actual: []runtime.Object{
				&corev1.PersistentVolume{
					TypeMeta: metav1.TypeMeta{
						Kind:       "PersistentVolume",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:   "my-persistentvolume",
						Labels: labeling.ManageResource.New(),
					},
					Spec: corev1.PersistentVolumeSpec{
						PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRecycle,
					},
				},
			},
			wantDeletes: []runtime.Object{
				&corev1.PersistentVolume{
					TypeMeta: metav1.TypeMeta{
						Kind:       "PersistentVolume",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:   "my-persistentvolume",
						Labels: labeling.ManageResource.New(),
					},
					Spec: corev1.PersistentVolumeSpec{
						PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRecycle,
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
			name: "ignore clusterpolicy with invalid name",
			clusterPolicy: &v1.ClusterPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "some-incorrect-name",
				},
				Status: v1.ClusterPolicyStatus{
					SyncState: v1.StateSynced,
				},
			},
			declared: []runtime.Object{},
			wantStatusUpdate: &v1.ClusterPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "some-incorrect-name",
				},
				Status: v1.ClusterPolicyStatus{
					SyncState: v1.StateError,
					SyncTime:  now(),
					SyncErrors: []v1.ClusterPolicySyncError{
						{
							ResourceName: "some-incorrect-name",
							ErrorMessage: `ClusterPolicy resource has invalid name "some-incorrect-name"`,
						},
					},
				},
			},
			wantEvents: []event{
				{
					kind:    corev1.EventTypeWarning,
					reason:  "InvalidClusterPolicy",
					varargs: true,
				},
			},
		},
	}

	converter := runtime.NewTestUnstructuredConverter(conversion.EqualitiesOrDie())
	gvk := schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "PersistentVolume",
	}
	comparator := syncerdiffer.NewComparator([]*v1alpha1.Sync{sync(gvk)}, labeling.ResourceManagementKey)
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

			testReconciler := NewClusterPolicyReconciler(
				client.New(mockClient), mockApplier, mockCache, mockRecorder, fakeDecoder, comparator, toSync)

			// Get ClusterPolicy from cache.
			mockCache.EXPECT().
				Get(gomock.Any(), gomock.Any(), gomock.Any()).
				SetArg(2, *tc.clusterPolicy)

			// List actual resources on the cluster.
			if tc.actual != nil {
				mockCache.EXPECT().
					UnstructuredList(gomock.Any(), gomock.Eq("")).
					Return(toUnstructureds(t, converter, tc.actual), nil)
			}

			// Check for expected creates, applies and deletes.
			for _, wantCreate := range tc.wantCreates {
				mockApplier.EXPECT().
					Create(gomock.Any(), NewUnstructuredMatcher(toUnstructured(t, converter, wantCreate)))
			}
			for _, wantApply := range tc.wantApplies {
				mockApplier.EXPECT().
					ApplyCluster(
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
						Name: tc.clusterPolicy.Name,
					},
				})
			if err != nil {
				t.Errorf("unexpected reconciliation error: %v", err)
			}
		})
	}
}
