package crd

import (
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/metrics"
	"github.com/google/nomos/pkg/syncer/sync"
	"github.com/google/nomos/pkg/syncer/syncertest"
	testingfake "github.com/google/nomos/pkg/syncer/syncertest/fake"
	"github.com/google/nomos/pkg/testing/fake"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	clusterReconcileComplete = *testingfake.NewEvent(
		fake.CRDClusterConfigObject(), corev1.EventTypeNormal, v1.EventReasonReconcileComplete)

	crdUpdated = *testingfake.NewEvent(
		fake.CRDClusterConfigObject(), corev1.EventTypeNormal, v1.EventReasonCRDChange)
)

func clusterConfig(state v1.ConfigSyncState, opts ...fake.ClusterConfigMutator) *v1.ClusterConfig {
	result := fake.ClusterConfigObject(opts...)
	result.Status.SyncState = state
	return result
}

func customResourceDefinitionV1Beta1(version string, opts ...core.MetaMutator) *v1beta1.CustomResourceDefinition {
	result := fake.CustomResourceDefinitionV1Beta1Object(opts...)
	result.Spec.Versions = []v1beta1.CustomResourceDefinitionVersion{{Name: version}}
	return result
}

func crdList(gvks []schema.GroupVersionKind) []v1beta1.CustomResourceDefinition {
	crdSpecs := map[schema.GroupKind][]string{}
	for _, gvk := range gvks {
		gk := gvk.GroupKind()
		crdSpecs[gk] = append(crdSpecs[gk], gvk.Version)
	}
	var crdList []v1beta1.CustomResourceDefinition
	for gk, vers := range crdSpecs {
		crd := v1beta1.CustomResourceDefinition{
			TypeMeta: metav1.TypeMeta{
				APIVersion: kinds.CustomResourceDefinitionV1Beta1().GroupVersion().String(),
				Kind:       kinds.CustomResourceDefinitionV1Beta1().Kind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: strings.ToLower(gk.Kind + "s." + gk.Group),
			},
		}
		crd.Spec.Group = gk.Group
		crd.Spec.Names.Kind = gk.Kind
		for _, ver := range vers {
			crd.Spec.Versions = append(crd.Spec.Versions, v1beta1.CustomResourceDefinitionVersion{
				Name: ver,
			})
		}
		crdList = append(crdList, crd)
	}
	return crdList
}

var (
	clusterCfg = clusterConfig(v1.StateSynced,
		syncertest.ClusterConfigImportToken(syncertest.Token),
		syncertest.ClusterConfigImportTime(metav1.NewTime(syncertest.Now().Add(time.Minute))),
		fake.ClusterConfigMeta(core.Name(v1.CRDClusterConfigName)),
	)

	clusterCfgSynced = clusterConfig(v1.StateSynced,
		syncertest.ClusterConfigImportToken(syncertest.Token),
		syncertest.ClusterConfigImportTime(metav1.NewTime(syncertest.Now().Add(time.Minute))),
		fake.ClusterConfigMeta(core.Name(v1.CRDClusterConfigName)),
		syncertest.ClusterConfigSyncTime(),
		syncertest.ClusterConfigSyncToken(),
	)
)

type crdTestCase struct {
	name     string
	actual   runtime.Object
	declared runtime.Object
	// initialCrds is the list of CRDs on the reconciler at start
	initialCrds []schema.GroupVersionKind
	// listCrds if the list of CRDs on the API Server
	listCrds      []schema.GroupVersionKind
	want          []runtime.Object
	expectEvents  []testingfake.Event
	expectRestart bool
}

func TestClusterConfigReconcile(t *testing.T) {
	testCases := []crdTestCase{
		{
			name:     "create from declared state",
			declared: customResourceDefinitionV1Beta1(v1Version),
			want: []runtime.Object{
				clusterCfgSynced,
				customResourceDefinitionV1Beta1(v1Version,
					syncertest.TokenAnnotation, syncertest.ManagementEnabled),
			},
			expectEvents:  []testingfake.Event{clusterReconcileComplete, crdUpdated},
			expectRestart: true,
		},
		{
			name:     "do not create if management disabled",
			declared: customResourceDefinitionV1Beta1(v1Version, syncertest.ManagementDisabled),
			want: []runtime.Object{
				clusterCfgSynced,
			},
		},
		// The declared state is invalid, so take no action.
		{
			name:     "do not create if management invalid",
			declared: customResourceDefinitionV1Beta1(v1Version, syncertest.ManagementInvalid),
			want: []runtime.Object{
				clusterCfgSynced,
			},
			expectEvents: []testingfake.Event{
				*testingfake.NewEvent(fake.CustomResourceDefinitionV1Beta1Object(), corev1.EventTypeWarning, v1.EventReasonInvalidAnnotation),
			},
		},
		{
			name:     "update to declared state",
			declared: customResourceDefinitionV1Beta1(v1Version),
			actual:   customResourceDefinitionV1Beta1(v1beta1Version, syncertest.ManagementEnabled),
			want: []runtime.Object{
				clusterCfgSynced,
				customResourceDefinitionV1Beta1(v1Version, syncertest.TokenAnnotation, syncertest.ManagementEnabled),
			},
			expectEvents:  []testingfake.Event{clusterReconcileComplete, crdUpdated},
			expectRestart: true,
		},
		{
			name:     "update to declared state even if actual managed unset",
			declared: customResourceDefinitionV1Beta1(v1Version),
			actual:   customResourceDefinitionV1Beta1(v1beta1Version),
			want: []runtime.Object{
				clusterCfgSynced,
				customResourceDefinitionV1Beta1(v1Version, syncertest.TokenAnnotation, syncertest.ManagementEnabled),
			},
			expectEvents:  []testingfake.Event{clusterReconcileComplete, crdUpdated},
			expectRestart: true,
		},
		// The declared state is fine, so overwrite the invalid one on the API Server.
		{
			name:     "update to declared state if actual managed invalid",
			declared: customResourceDefinitionV1Beta1(v1Version),
			actual:   customResourceDefinitionV1Beta1(v1beta1Version, syncertest.ManagementInvalid),
			want: []runtime.Object{
				clusterCfgSynced,
				customResourceDefinitionV1Beta1(v1Version, syncertest.TokenAnnotation, syncertest.ManagementEnabled),
			},
			expectEvents:  []testingfake.Event{clusterReconcileComplete, crdUpdated},
			expectRestart: true,
		},
		// The declared state is invalid, so take no action.
		{
			name:     "do not update if declared management invalid",
			declared: customResourceDefinitionV1Beta1(v1Version, syncertest.ManagementInvalid),
			actual:   customResourceDefinitionV1Beta1(v1beta1Version),
			initialCrds: []schema.GroupVersionKind{
				{Group: "", Version: v1beta1Version, Kind: ""},
			},
			want: []runtime.Object{
				clusterCfgSynced,
				customResourceDefinitionV1Beta1(v1beta1Version),
			},
			expectEvents: []testingfake.Event{
				*testingfake.NewEvent(fake.CustomResourceDefinitionV1Beta1Object(), corev1.EventTypeWarning, v1.EventReasonInvalidAnnotation),
			},
		},
		{
			name:     "update to unmanaged",
			declared: customResourceDefinitionV1Beta1(v1Version, syncertest.ManagementDisabled),
			actual:   customResourceDefinitionV1Beta1(v1beta1Version, syncertest.ManagementEnabled),
			want: []runtime.Object{
				clusterCfgSynced,
				customResourceDefinitionV1Beta1(v1beta1Version),
			},
			expectEvents:  []testingfake.Event{clusterReconcileComplete, crdUpdated},
			expectRestart: true,
		},
		{
			name:     "do not update if unmanaged",
			declared: customResourceDefinitionV1Beta1(v1Version, syncertest.ManagementDisabled),
			actual:   customResourceDefinitionV1Beta1(v1beta1Version),
			initialCrds: []schema.GroupVersionKind{
				{Group: "", Version: v1beta1Version, Kind: ""},
			},
			want: []runtime.Object{
				clusterCfgSynced,
				customResourceDefinitionV1Beta1(v1beta1Version),
			},
		},
		{
			name:   "delete if managed",
			actual: customResourceDefinitionV1Beta1(v1beta1Version, syncertest.ManagementEnabled),
			want: []runtime.Object{
				clusterCfgSynced,
			},
			expectEvents:  []testingfake.Event{clusterReconcileComplete},
			expectRestart: true,
		},
		{
			name:   "do not delete if unmanaged",
			actual: customResourceDefinitionV1Beta1(v1beta1Version),
			initialCrds: []schema.GroupVersionKind{
				{Group: "", Version: v1beta1Version, Kind: ""},
			},
			want: []runtime.Object{
				clusterCfgSynced,
				customResourceDefinitionV1Beta1(v1beta1Version),
			},
		},
		// There is no declared state, just an invalid annotation.
		// This was most likely put there by a user, so remove it.
		{
			name:   "unmanage if invalid",
			actual: customResourceDefinitionV1Beta1(v1beta1Version, syncertest.ManagementInvalid),
			initialCrds: []schema.GroupVersionKind{
				{Group: "", Version: v1beta1Version, Kind: ""},
			},
			want: []runtime.Object{
				clusterCfgSynced,
				customResourceDefinitionV1Beta1(v1beta1Version),
			},
			expectEvents:  []testingfake.Event{clusterReconcileComplete},
			expectRestart: true,
		},
		{
			name: "resource with owner reference is ignored",
			actual: customResourceDefinitionV1Beta1(v1Version, syncertest.ManagementEnabled,
				fake.OwnerReference(
					"some_operator_config_object",
					schema.GroupVersionKind{Group: "operator.config.group", Kind: "OperatorConfigObject", Version: v1Version}),
			),
			initialCrds: []schema.GroupVersionKind{
				{Group: "", Version: v1Version, Kind: ""},
			},
			want: []runtime.Object{
				clusterCfgSynced,
				customResourceDefinitionV1Beta1(v1Version, syncertest.ManagementEnabled,
					fake.OwnerReference(
						"some_operator_config_object",
						schema.GroupVersionKind{Group: "operator.config.group", Kind: "OperatorConfigObject", Version: v1Version}),
				),
			},
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
			want: []runtime.Object{
				clusterCfgSynced,
				customResourceDefinitionV1Beta1(v1Version,
					syncertest.TokenAnnotation, syncertest.ManagementEnabled),
			},
			expectEvents:  []testingfake.Event{clusterReconcileComplete, crdUpdated},
			expectRestart: true,
		},
		{
			name: "external crd change triggers restart",
			want: []runtime.Object{
				clusterCfgSynced,
			},
			expectEvents: []testingfake.Event{crdUpdated},
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
			name: "no change",
			want: []runtime.Object{
				clusterCfgSynced,
			},
			initialCrds: []schema.GroupVersionKind{
				{Group: "foo.xyz", Version: "v1", Kind: "Stuff"},
			},
			listCrds: []schema.GroupVersionKind{
				{Group: "foo.xyz", Version: "v1", Kind: "Stuff"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name+" v1beta1", tc.run)

		// Convert the test case's v1beta1 CRDs to v1 CRDs.
		if tc.declared != nil {
			tc.declared = fake.ToCustomResourceDefinitionV1Object(tc.declared.(*v1beta1.CustomResourceDefinition))
		}
		if tc.actual != nil {
			tc.actual = fake.ToCustomResourceDefinitionV1Object(tc.actual.(*v1beta1.CustomResourceDefinition))
		}
		for i, o := range tc.want {
			if o.GetObjectKind().GroupVersionKind() == kinds.CustomResourceDefinitionV1Beta1() {
				tc.want[i] = fake.ToCustomResourceDefinitionV1Object(tc.want[i].(*v1beta1.CustomResourceDefinition))
			}
		}

		for i, event := range tc.expectEvents {
			if event.GroupVersionKind == kinds.CustomResourceDefinitionV1Beta1() {
				// Only change event object type if it is a v1beta1.CRD.
				tc.expectEvents[i].GroupVersionKind = kinds.CustomResourceDefinitionV1()
			}
		}

		t.Run(tc.name+" v1", tc.run)
	}
}

func (tc crdTestCase) run(t *testing.T) {
	fakeDecoder := testingfake.NewDecoder(syncertest.ToUnstructuredList(t, syncertest.Converter, tc.declared))
	fakeEventRecorder := testingfake.NewEventRecorder(t)
	fakeSignal := RestartSignalRecorder{}
	actual := []runtime.Object{clusterCfg}
	if tc.actual != nil {
		actual = append(actual, tc.actual)
	}
	for _, crd := range crdList(tc.listCrds) {
		actual = append(actual, crd.DeepCopy())
	}

	s := runtime.NewScheme()
	s.AddKnownTypeWithName(kinds.CustomResourceDefinitionV1Beta1(), &v1beta1.CustomResourceDefinition{})
	// TODO(b/154527698): Replace v1beta1 stand-ins with v1.CustomResourceDefinitions.
	s.AddKnownTypeWithName(kinds.CustomResourceDefinitionV1(), &v1beta1.CustomResourceDefinition{})
	fakeClient := testingfake.NewClient(t, s, actual...)

	testReconciler := newReconciler(client.New(fakeClient, metrics.APICallDuration), fakeClient.Applier(), fakeClient, fakeEventRecorder,
		fakeDecoder, syncertest.Now, &fakeSignal)
	testReconciler.allCrds = testReconciler.toCrdSet(crdList(tc.initialCrds))

	_, err := testReconciler.Reconcile(
		runtimereconcile.Request{
			NamespacedName: types.NamespacedName{
				Name: v1.CRDClusterConfigName,
			},
		})

	if tc.expectRestart {
		fakeSignal.Check(t, restartSignal)
	} else {
		fakeSignal.Check(t)
	}
	fakeEventRecorder.Check(t, tc.expectEvents...)

	want := tc.want
	for _, crd := range crdList(tc.listCrds) {
		want = append(want, crd.DeepCopy())
	}
	fakeClient.Check(t, want...)
	if err != nil {
		t.Errorf("unexpected reconciliation error: %v", err)
	}
}

// RestartSignalRecorder implements a fake sync.RestartSignal.
type RestartSignalRecorder struct {
	Restarts []string
}

var _ sync.RestartSignal = &RestartSignalRecorder{}

// Restart implements RestartSignal.
func (r *RestartSignalRecorder) Restart(signal string) {
	r.Restarts = append(r.Restarts, signal)
}

// Check ensures that the RestartSignal was called exactly with the passed
// sequence of signals.
func (r *RestartSignalRecorder) Check(t *testing.T, want ...string) {
	if diff := cmp.Diff(want, r.Restarts); diff != "" {
		t.Errorf("Diff in calls to fake.RestartSignalRecorder.Restart(): %s", diff)
	}
}
