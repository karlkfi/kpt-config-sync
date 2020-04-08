package crd

import (
	"testing"

	"github.com/golang/mock/gomock"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/metrics"
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

func customResourceDefinitionV1Beta1(version string, opts ...core.MetaMutator) *v1beta1.CustomResourceDefinition {
	mutators := append(opts, syncertesting.Herrings...)
	result := fake.CustomResourceDefinitionV1Beta1Object(mutators...)
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

type crdTestCase struct {
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
}

func TestClusterConfigReconcile(t *testing.T) {
	testCases := []crdTestCase{
		{
			name:     "create from declared state",
			declared: customResourceDefinitionV1Beta1(v1Version),
			expectCreate: customResourceDefinitionV1Beta1(v1Version, syncertesting.TokenAnnotation,
				syncertesting.ManagementEnabled),
			expectStatusUpdate: clusterCfgSynced,
			expectEvent:        clusterReconcileComplete,
			expectRestart:      true,
		},
		{
			name:               "do not create if management disabled",
			declared:           customResourceDefinitionV1Beta1(v1Version, syncertesting.ManagementDisabled),
			expectStatusUpdate: clusterCfgSynced,
		},
		{
			name:               "do not create if management invalid",
			declared:           customResourceDefinitionV1Beta1(v1Version, syncertesting.ManagementInvalid),
			expectStatusUpdate: clusterCfgSynced,
			expectEvent: &syncertesting.Event{
				Kind:   corev1.EventTypeWarning,
				Reason: "InvalidAnnotation",
				Obj: syncertesting.ToUnstructured(t, syncertesting.Converter, customResourceDefinitionV1Beta1(v1Version, syncertesting.ManagementInvalid,
					syncertesting.TokenAnnotation)),
			},
		},
		{
			name:     "update to declared state",
			declared: customResourceDefinitionV1Beta1(v1Version),
			actual:   customResourceDefinitionV1Beta1(v1beta1Version, syncertesting.ManagementEnabled),
			expectUpdate: &syncertesting.Diff{
				Declared: customResourceDefinitionV1Beta1(v1Version, syncertesting.TokenAnnotation, syncertesting.ManagementEnabled),
				Actual:   customResourceDefinitionV1Beta1(v1beta1Version, syncertesting.ManagementEnabled),
			},
			expectStatusUpdate: clusterCfgSynced,
			expectEvent:        clusterReconcileComplete,
			expectRestart:      true,
		},
		{
			name:     "update to declared state even if actual managed unset",
			declared: customResourceDefinitionV1Beta1(v1Version),
			actual:   customResourceDefinitionV1Beta1(v1beta1Version),
			expectUpdate: &syncertesting.Diff{
				Declared: customResourceDefinitionV1Beta1(v1Version, syncertesting.TokenAnnotation, syncertesting.ManagementEnabled),
				Actual:   customResourceDefinitionV1Beta1(v1beta1Version),
			},
			expectStatusUpdate: clusterCfgSynced,
			expectEvent:        clusterReconcileComplete,
			expectRestart:      true,
		},
		{
			name:     "update to declared state even if actual managed invalid",
			declared: customResourceDefinitionV1Beta1(v1Version),
			actual:   customResourceDefinitionV1Beta1(v1beta1Version, syncertesting.ManagementInvalid),
			expectUpdate: &syncertesting.Diff{
				Declared: customResourceDefinitionV1Beta1(v1Version, syncertesting.TokenAnnotation, syncertesting.ManagementEnabled),
				Actual:   customResourceDefinitionV1Beta1(v1beta1Version, syncertesting.ManagementInvalid),
			},
			expectStatusUpdate: clusterCfgSynced,
			expectEvent:        clusterReconcileComplete,
			expectRestart:      true,
		},
		{
			name:               "do not update if declared management invalid",
			declared:           customResourceDefinitionV1Beta1(v1Version, syncertesting.ManagementInvalid),
			actual:             customResourceDefinitionV1Beta1(v1beta1Version),
			expectStatusUpdate: clusterCfgSynced,
			expectEvent: &syncertesting.Event{
				Kind:   corev1.EventTypeWarning,
				Reason: "InvalidAnnotation",
				Obj: syncertesting.ToUnstructured(t, syncertesting.Converter, customResourceDefinitionV1Beta1(v1Version, syncertesting.ManagementInvalid,
					syncertesting.TokenAnnotation)),
			},
		},
		{
			name:     "update to unmanaged",
			declared: customResourceDefinitionV1Beta1(v1Version, syncertesting.ManagementDisabled),
			actual:   customResourceDefinitionV1Beta1(v1beta1Version, syncertesting.ManagementEnabled),
			expectUpdate: &syncertesting.Diff{
				Declared: customResourceDefinitionV1Beta1(v1beta1Version),
				Actual:   customResourceDefinitionV1Beta1(v1beta1Version, syncertesting.ManagementEnabled),
			},
			expectStatusUpdate: clusterCfgSynced,
			expectEvent:        clusterReconcileComplete,
			expectRestart:      true,
		},
		{
			name:               "do not update if unmanaged",
			declared:           customResourceDefinitionV1Beta1(v1Version, syncertesting.ManagementDisabled),
			actual:             customResourceDefinitionV1Beta1(v1beta1Version),
			expectStatusUpdate: clusterCfgSynced,
		},
		{
			name:               "delete if managed",
			actual:             customResourceDefinitionV1Beta1(v1beta1Version, syncertesting.ManagementEnabled),
			expectDelete:       customResourceDefinitionV1Beta1(v1beta1Version, syncertesting.ManagementEnabled),
			expectStatusUpdate: clusterCfgSynced,
			expectEvent:        clusterReconcileComplete,
			expectRestart:      true,
		},
		{
			name:               "do not delete if unmanaged",
			actual:             customResourceDefinitionV1Beta1(v1beta1Version),
			expectStatusUpdate: clusterCfgSynced,
		},
		{
			name:   "unmanage if invalid",
			actual: customResourceDefinitionV1Beta1(v1beta1Version, syncertesting.ManagementInvalid),
			expectUpdate: &syncertesting.Diff{
				Declared: customResourceDefinitionV1Beta1(v1beta1Version),
				Actual:   customResourceDefinitionV1Beta1(v1beta1Version, syncertesting.ManagementInvalid),
			},
			expectStatusUpdate: clusterCfgSynced,
			expectEvent:        clusterReconcileComplete,
			expectRestart:      true,
		},
		{
			name: "resource with owner reference is ignored",
			actual: customResourceDefinitionV1Beta1(v1Version, syncertesting.ManagementEnabled,
				fake.OwnerReference(
					"some_operator_config_object",
					schema.GroupVersionKind{Group: "operator.config.group", Kind: "OperatorConfigObject", Version: v1Version}),
			),
			expectStatusUpdate: clusterCfgSynced,
		},
		{
			name:     "create from declared state and external crd change",
			declared: customResourceDefinitionV1Beta1(v1Version),
			initialCrds: []schema.GroupVersionKind{
				{Group: "foo.xyz", Version: "v1", Kind: "Stuff"},
			},
			listCrds: []schema.GroupVersionKind{
				{Group: "foo.xyz", Version: "v1", Kind: "Stuff"},
				{Group: "bar.xyz", Version: "v1", Kind: "MoreStuff"},
			},
			expectCreate: customResourceDefinitionV1Beta1(v1Version, syncertesting.TokenAnnotation,
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
			tc.run(t)

			if tc.declared != nil {
				tc.declared = fake.ToCustomResourceDefinitionV1Object(tc.declared.(*v1beta1.CustomResourceDefinition))
			}
			if tc.actual != nil {
				tc.actual = fake.ToCustomResourceDefinitionV1Object(tc.actual.(*v1beta1.CustomResourceDefinition))
			}
			if tc.expectCreate != nil {
				tc.expectCreate = fake.ToCustomResourceDefinitionV1Object(tc.expectCreate.(*v1beta1.CustomResourceDefinition))
			}
			if tc.expectUpdate != nil {
				if tc.expectUpdate.Actual != nil {
					tc.expectUpdate.Actual = fake.ToCustomResourceDefinitionV1Object(tc.expectUpdate.Actual.(*v1beta1.CustomResourceDefinition))
				}
				if tc.expectUpdate.Declared != nil {
					tc.expectUpdate.Declared = fake.ToCustomResourceDefinitionV1Object(tc.expectUpdate.Declared.(*v1beta1.CustomResourceDefinition))
				}
			}
			if tc.expectDelete != nil {
				tc.expectDelete = fake.ToCustomResourceDefinitionV1Object(tc.expectDelete.(*v1beta1.CustomResourceDefinition))
			}

			if tc.expectEvent != nil && tc.expectEvent.Obj != nil && tc.expectEvent.Obj.GetObjectKind().GroupVersionKind() == kinds.CustomResourceDefinitionV1Beta1() {
				// Only change event object if it is a v1beta1.CRD.
				tc.expectEvent.Obj.GetObjectKind().SetGroupVersionKind(kinds.CustomResourceDefinitionV1())
			}
			if tc.expectEventAux != nil && tc.expectEventAux.Obj != nil && tc.expectEvent.Obj.GetObjectKind().GroupVersionKind() == kinds.CustomResourceDefinitionV1Beta1() {
				// Only change aux event object if it is a v1beta1.CRD.
				tc.expectEventAux.Obj.GetObjectKind().SetGroupVersionKind(kinds.CustomResourceDefinitionV1())
			}

			tc.run(t)
		})
	}
}

func (tc crdTestCase) run(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tm := syncertesting.NewTestMocks(t, mockCtrl)
	fakeDecoder := mocks.NewFakeDecoder(syncertesting.ToUnstructuredList(t, syncertesting.Converter, tc.declared))
	testReconciler := NewReconciler(client.New(tm.MockClient, metrics.APICallDuration), tm.MockApplier, tm.MockCache, tm.MockRecorder,
		fakeDecoder, syncertesting.Now, tm.MockSignal)
	testReconciler.allCrds = testReconciler.toCrdSet(crdList(tc.initialCrds).Items)

	tm.ExpectClusterCacheGet(clusterCfg)
	tm.ExpectCacheList(kinds.CustomResourceDefinitionV1Beta1(), "", tc.actual)
	if tc.declared != nil && tc.declared.GetObjectKind().GroupVersionKind() == kinds.CustomResourceDefinitionV1() {
		tm.ExpectCacheList(kinds.CustomResourceDefinitionV1(), "", tc.actual)
	}

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
}
