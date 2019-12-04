package crd

import (
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
	externalCRDUpdated = &syncertesting.Event{
		Kind:    corev1.EventTypeNormal,
		Reason:  "CRDChange",
		Varargs: true,
		Obj:     clusterCfg,
	}
)

func clusterConfig(state v1.ConfigSyncState, opts ...fake.ClusterConfigMutator) *v1.ClusterConfig {
	mutators := append(opts, fake.ClusterConfigMeta(syncertesting.Herrings...))
	result := fake.ClusterConfigObject(mutators...)
	result.Status.SyncState = state
	return result
}

func customResourceDefinition(version string, opts ...core.MetaMutator) *v1beta1.CustomResourceDefinition {
	mutators := append(opts, syncertesting.Herrings...)
	result := fake.CustomResourceDefinitionObject(mutators...)
	result.Spec.Versions = []v1beta1.CustomResourceDefinitionVersion{{Name: version}}
	return result
}

func crdList(gvks []schema.GroupVersionKind) v1beta1.CustomResourceDefinitionList {
	crdSpecs := map[schema.GroupKind][]string{}
	for _, gvk := range gvks {
		gk := gvk.GroupKind()
		crdSpecs[gk] = append(crdSpecs[gk], gvk.Version)
	}
	var crdList v1beta1.CustomResourceDefinitionList
	for gk, vers := range crdSpecs {
		var crd v1beta1.CustomResourceDefinition
		crd.Spec.Group = gk.Group
		crd.Spec.Names.Kind = gk.Kind
		for _, ver := range vers {
			crd.Spec.Versions = append(crd.Spec.Versions, v1beta1.CustomResourceDefinitionVersion{
				Name: ver,
			})
		}
		crdList.Items = append(crdList.Items, crd)
	}
	return crdList
}

var (
	clusterCfg       = clusterConfig(v1.StateSynced, syncertesting.ClusterConfigImportToken(syncertesting.Token), fake.ClusterConfigMeta(core.Name(v1.CRDClusterConfigName)))
	clusterCfgSynced = clusterConfig(v1.StateSynced, syncertesting.ClusterConfigImportToken(syncertesting.Token),
		fake.ClusterConfigMeta(core.Name(v1.CRDClusterConfigName)), syncertesting.ClusterConfigSyncTime(), syncertesting.ClusterConfigSyncToken())
)

func TestClusterConfigReconcile(t *testing.T) {
	testCases := []struct {
		name               string
		actual             runtime.Object
		declared           runtime.Object
		initialCrds        []schema.GroupVersionKind
		listCrds           []schema.GroupVersionKind
		expectCreate       runtime.Object
		expectUpdate       *syncertesting.Diff
		expectDelete       runtime.Object
		expectStatusUpdate *v1.ClusterConfig
		expectEvent        *syncertesting.Event
		expectEventAux     *syncertesting.Event
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
				fake.OwnerReference(
					"some_operator_config_object",
					schema.GroupVersionKind{Group: "operator.config.group", Kind: "OperatorConfigObject", Version: v1Version}),
			),
			expectStatusUpdate: clusterCfgSynced,
		},
		{
			name:     "create from declared state and external crd change",
			declared: customResourceDefinition(v1Version),
			initialCrds: []schema.GroupVersionKind{
				{Group: "foo.xyz", Version: "v1", Kind: "Stuff"},
			},
			listCrds: []schema.GroupVersionKind{
				{Group: "foo.xyz", Version: "v1", Kind: "Stuff"},
				{Group: "bar.xyz", Version: "v1", Kind: "MoreStuff"},
			},
			expectCreate: customResourceDefinition(v1Version, syncertesting.TokenAnnotation,
				syncertesting.ManagementEnabled),
			expectStatusUpdate: clusterCfgSynced,
			expectEvent:        clusterReconcileComplete,
			expectEventAux:     externalCRDUpdated,
			expectRestart:      true,
		},
		{
			name:               "external crd change triggers restart",
			expectStatusUpdate: clusterCfgSynced,
			expectEvent:        externalCRDUpdated,
			initialCrds: []schema.GroupVersionKind{
				{Group: "foo.xyz", Version: "v1", Kind: "Stuff"},
			},
			listCrds: []schema.GroupVersionKind{
				{Group: "foo.xyz", Version: "v1", Kind: "Stuff"},
				{Group: "bar.xyz", Version: "v1", Kind: "MoreStuff"},
			},
			expectRestart: true,
		},
		{
			name:               "no change",
			expectStatusUpdate: clusterCfgSynced,
			initialCrds: []schema.GroupVersionKind{
				{Group: "foo.xyz", Version: "v1", Kind: "Stuff"},
			},
			listCrds: []schema.GroupVersionKind{
				{Group: "foo.xyz", Version: "v1", Kind: "Stuff"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			tm := syncertesting.NewTestMocks(t, mockCtrl)
			fakeDecoder := mocks.NewFakeDecoder(syncertesting.ToUnstructuredList(t, syncertesting.Converter, tc.declared))
			testReconciler := NewReconciler(client.New(tm.MockClient, metrics.APICallDuration), tm.MockApplier, tm.MockCache, tm.MockRecorder,
				fakeDecoder, syncertesting.Now, tm.MockSignal)
			testReconciler.allCrds = testReconciler.toCrdSet(crdList(tc.initialCrds).Items)

			tm.ExpectClusterCacheGet(clusterCfg)
			tm.ExpectCacheList(kinds.CustomResourceDefinition(), "", tc.actual)

			tm.ExpectCreate(tc.expectCreate)
			tm.ExpectUpdate(tc.expectUpdate)
			tm.ExpectDelete(tc.expectDelete)

			call := tm.MockClient.EXPECT().List(gomock.Any(), &v1beta1.CustomResourceDefinitionList{}, gomock.Any()).Return(nil)
			if len(tc.listCrds) != 0 {
				call.SetArg(1, crdList(tc.listCrds))
			}

			tm.ExpectClusterClientGet(clusterCfg)
			tm.ExpectClusterStatusUpdate(tc.expectStatusUpdate)
			tm.ExpectEvent(tc.expectEvent)
			tm.ExpectEvent(tc.expectEventAux)
			tm.ExpectRestart(tc.expectRestart, "crd")

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
	}
}
