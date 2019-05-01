package crd

import (
	"testing"

	"github.com/google/nomos/pkg/syncer/metrics"

	"github.com/golang/mock/gomock"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/syncer/client"
	syncertesting "github.com/google/nomos/pkg/syncer/testing"
	"github.com/google/nomos/pkg/syncer/testing/mocks"
	"github.com/google/nomos/pkg/testing/fake"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	runtimereconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	v1Version      = "v1"
	v1beta1Version = "v1beta1"
)

var (
	clusterReconcileComplete = &syncertesting.Event{
		Kind:    corev1.EventTypeNormal,
		Reason:  "ReconcileComplete",
		Varargs: true,
		Obj:     clusterCfg,
	}
)

func clusterConfig(state v1.PolicySyncState, opts ...object.Mutator) *v1.ClusterConfig {
	opts = append(opts, func(o *ast.FileObject) {
		o.Object.(*v1.ClusterConfig).Status.SyncState = state
	})
	return fake.Build(kinds.ClusterConfig(), opts...).Object.(*v1.ClusterConfig)
}

func customResourceDefinition(version string, opts ...object.Mutator) *v1beta1.CustomResourceDefinition {
	opts = append(opts, func(o *ast.FileObject) {
		o.Object.(*v1beta1.CustomResourceDefinition).Spec.Versions = []v1beta1.CustomResourceDefinitionVersion{{Name: version}}
	}, syncertesting.Herrings)
	return fake.Build(kinds.CustomResourceDefinition(), opts...).Object.(*v1beta1.CustomResourceDefinition)
}

var (
	clusterCfg       = clusterConfig(v1.StateSynced, syncertesting.ImportToken(syncertesting.Token), object.Name(v1.CRDClusterConfigName))
	clusterCfgSynced = clusterConfig(v1.StateSynced, syncertesting.ImportToken(syncertesting.Token),
		object.Name(v1.CRDClusterConfigName), syncertesting.SyncTime(), syncertesting.SyncToken())
)

func TestClusterConfigReconcile(t *testing.T) {
	testCases := []struct {
		name               string
		actual             runtime.Object
		declared           runtime.Object
		expectCreate       runtime.Object
		expectUpdate       *syncertesting.Diff
		expectDelete       runtime.Object
		expectStatusUpdate *v1.ClusterConfig
		expectEvent        *syncertesting.Event
		expectRestart      bool
	}{
		{
			name:     "create from declared state",
			declared: customResourceDefinition(v1Version),
			expectCreate: customResourceDefinition(v1Version, syncertesting.TokenAnnotation,
				syncertesting.ManagementEnabled),
			expectStatusUpdate: clusterCfgSynced,
			expectEvent:        clusterReconcileComplete,
			expectRestart:      true,
		},
		{
			name:               "do not create if management disabled",
			declared:           customResourceDefinition(v1Version, syncertesting.ManagementDisabled),
			expectStatusUpdate: clusterCfgSynced,
		},
		{
			name:               "do not create if management invalid",
			declared:           customResourceDefinition(v1Version, syncertesting.ManagementInvalid),
			expectStatusUpdate: clusterCfgSynced,
			expectEvent: &syncertesting.Event{
				Kind:   corev1.EventTypeWarning,
				Reason: "InvalidAnnotation",
				Obj: syncertesting.ToUnstructured(t, syncertesting.Converter, customResourceDefinition(v1Version, syncertesting.ManagementInvalid,
					syncertesting.TokenAnnotation)),
			},
		},
		{
			name:     "update to declared state",
			declared: customResourceDefinition(v1Version),
			actual:   customResourceDefinition(v1beta1Version, syncertesting.ManagementEnabled),
			expectUpdate: &syncertesting.Diff{
				Declared: customResourceDefinition(v1Version, syncertesting.TokenAnnotation, syncertesting.ManagementEnabled),
				Actual:   customResourceDefinition(v1beta1Version, syncertesting.ManagementEnabled),
			},
			expectStatusUpdate: clusterCfgSynced,
			expectEvent:        clusterReconcileComplete,
			expectRestart:      true,
		},
		{
			name:     "update to declared state even if actual managed unset",
			declared: customResourceDefinition(v1Version),
			actual:   customResourceDefinition(v1beta1Version),
			expectUpdate: &syncertesting.Diff{
				Declared: customResourceDefinition(v1Version, syncertesting.TokenAnnotation, syncertesting.ManagementEnabled),
				Actual:   customResourceDefinition(v1beta1Version),
			},
			expectStatusUpdate: clusterCfgSynced,
			expectEvent:        clusterReconcileComplete,
			expectRestart:      true,
		},
		{
			name:     "update to declared state even if actual managed invalid",
			declared: customResourceDefinition(v1Version),
			actual:   customResourceDefinition(v1beta1Version, syncertesting.ManagementInvalid),
			expectUpdate: &syncertesting.Diff{
				Declared: customResourceDefinition(v1Version, syncertesting.TokenAnnotation, syncertesting.ManagementEnabled),
				Actual:   customResourceDefinition(v1beta1Version, syncertesting.ManagementInvalid),
			},
			expectStatusUpdate: clusterCfgSynced,
			expectEvent:        clusterReconcileComplete,
			expectRestart:      true,
		},
		{
			name:               "do not update if declared management invalid",
			declared:           customResourceDefinition(v1Version, syncertesting.ManagementInvalid),
			actual:             customResourceDefinition(v1beta1Version),
			expectStatusUpdate: clusterCfgSynced,
			expectEvent: &syncertesting.Event{
				Kind:   corev1.EventTypeWarning,
				Reason: "InvalidAnnotation",
				Obj: syncertesting.ToUnstructured(t, syncertesting.Converter, customResourceDefinition(v1Version, syncertesting.ManagementInvalid,
					syncertesting.TokenAnnotation)),
			},
		},
		{
			name:     "update to unmanaged",
			declared: customResourceDefinition(v1Version, syncertesting.ManagementDisabled),
			actual:   customResourceDefinition(v1beta1Version, syncertesting.ManagementEnabled),
			expectUpdate: &syncertesting.Diff{
				Declared: customResourceDefinition(v1beta1Version),
				Actual:   customResourceDefinition(v1beta1Version, syncertesting.ManagementEnabled),
			},
			expectStatusUpdate: clusterCfgSynced,
			expectEvent:        clusterReconcileComplete,
			expectRestart:      true,
		},
		{
			name:               "do not update if unmanaged",
			declared:           customResourceDefinition(v1Version, syncertesting.ManagementDisabled),
			actual:             customResourceDefinition(v1beta1Version),
			expectStatusUpdate: clusterCfgSynced,
		},
		{
			name:               "delete if managed",
			actual:             customResourceDefinition(v1beta1Version, syncertesting.ManagementEnabled),
			expectDelete:       customResourceDefinition(v1beta1Version, syncertesting.ManagementEnabled),
			expectStatusUpdate: clusterCfgSynced,
			expectEvent:        clusterReconcileComplete,
			expectRestart:      true,
		},
		{
			name:               "do not delete if unmanaged",
			actual:             customResourceDefinition(v1beta1Version),
			expectStatusUpdate: clusterCfgSynced,
		},
		{
			name:   "unmanage if invalid",
			actual: customResourceDefinition(v1beta1Version, syncertesting.ManagementInvalid),
			expectUpdate: &syncertesting.Diff{
				Declared: customResourceDefinition(v1beta1Version),
				Actual:   customResourceDefinition(v1beta1Version, syncertesting.ManagementInvalid),
			},
			expectStatusUpdate: clusterCfgSynced,
			expectEvent:        clusterReconcileComplete,
			expectRestart:      true,
		},
		{
			name: "resource with owner reference is ignored",
			actual: customResourceDefinition(v1Version, syncertesting.ManagementEnabled,
				object.OwnerReference(
					"some_operator_config_object",
					"some_uid",
					schema.GroupVersionKind{Group: "operator.config.group", Kind: "OperatorConfigObject", Version: v1Version}),
			),
			expectStatusUpdate: clusterCfgSynced,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run(tc.name, func(t *testing.T) {
				mockCtrl := gomock.NewController(t)
				defer mockCtrl.Finish()

				tm := syncertesting.NewTestMocks(t, mockCtrl)
				fakeDecoder := mocks.NewFakeDecoder(syncertesting.ToUnstructuredList(t, syncertesting.Converter, tc.declared))
				testReconciler := NewReconciler(client.New(tm.MockClient, metrics.APICallDuration), tm.MockApplier, tm.MockCache, tm.MockRecorder,
					fakeDecoder, syncertesting.Now, tm.MockSignal)

				tm.ExpectClusterCacheGet(clusterCfg)
				tm.ExpectCacheList(kinds.CustomResourceDefinition(), "", tc.actual)

				tm.ExpectCreate(tc.expectCreate)
				tm.ExpectUpdate(tc.expectUpdate)
				tm.ExpectDelete(tc.expectDelete)

				tm.ExpectClusterClientGet(clusterCfg)
				tm.ExpectClusterStatusUpdate(tc.expectStatusUpdate)
				tm.ExpectEvent(tc.expectEvent)
				tm.ExpectRestart(tc.expectRestart)

				_, err := testReconciler.Reconcile(
					runtimereconcile.Request{
						NamespacedName: types.NamespacedName{
							Name: v1.CRDClusterConfigName,
						},
					})
				if err != nil {
					t.Errorf("unexpected reconciliation error: %v", err)
				}
			})
		})
	}
}
