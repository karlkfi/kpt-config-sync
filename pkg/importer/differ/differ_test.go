package differ

import (
	"context"
	"testing"
	"time"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/metrics"
	testingfake "github.com/google/nomos/pkg/syncer/syncertest/fake"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var testTime = metav1.NewTime(time.Unix(1234, 5678))

func namespaceConfig(name string, opts ...core.MetaMutator) *v1.NamespaceConfig {
	opts = append(opts, core.Name(name))
	return fake.NamespaceConfigObject(opts...)
}

func markedForDeletion(o core.Object) {
	nc := o.(*v1.NamespaceConfig)
	nc.Spec.DeleteSyncedTime = testTime
}

// allConfigs constructs a (potentially-invalid) AllConfigs for the purpose
// of Differ tests.
//
// Not intended for use with other tests - this is to make differ tests easy to
// specify, not replicate code that creates AllConfigs.
func allConfigs(t *testing.T, objs []runtime.Object) *namespaceconfig.AllConfigs {
	t.Helper()

	result := &namespaceconfig.AllConfigs{
		NamespaceConfigs: make(map[string]v1.NamespaceConfig),
		Syncs:            make(map[string]v1.Sync),
	}
	for _, o := range objs {
		switch obj := o.(type) {
		case *v1.NamespaceConfig:
			result.NamespaceConfigs[obj.Name] = *obj
		case *v1.Sync:
			result.Syncs[obj.Name] = *obj
		case *v1.ClusterConfig:
			switch obj.Name {
			case v1.ClusterConfigName:
				result.ClusterConfig = obj
			case v1.CRDClusterConfigName:
				result.CRDClusterConfig = obj
			default:
				t.Fatalf("unsupported ClusterConfig name %q", obj.Name)
			}
		default:
			t.Fatalf("unsupported AllConfigs type: %T", o)
		}
	}

	return result
}

func TestDiffer(t *testing.T) {
	// Mock out metav1.Now for testing.
	now = func() metav1.Time {
		return testTime
	}

	tcs := []struct {
		testName string
		actual   []runtime.Object
		declared []runtime.Object
		want     []runtime.Object
	}{
		// NamespaceConfig tests
		{
			testName: "create Namespace node",
			declared: []runtime.Object{namespaceConfig("foo")},
			want:     []runtime.Object{namespaceConfig("foo")},
		},
		{
			testName: "no-op Namespace node",
			actual:   []runtime.Object{namespaceConfig("foo")},
			declared: []runtime.Object{namespaceConfig("foo")},
			want:     []runtime.Object{namespaceConfig("foo")},
		},
		{
			testName: "update Namespace node",
			actual: []runtime.Object{namespaceConfig("foo",
				core.Annotation("key", "old"))},
			declared: []runtime.Object{namespaceConfig("foo",
				core.Annotation("key", "new"))},
			want: []runtime.Object{namespaceConfig("foo",
				core.Annotation("key", "new"))},
		},
		{
			testName: "delete Namespace node",
			actual:   []runtime.Object{namespaceConfig("foo")},
			want:     []runtime.Object{namespaceConfig("foo", markedForDeletion)},
		},
		{
			testName: "replace one Namespace node",
			actual:   []runtime.Object{namespaceConfig("foo")},
			declared: []runtime.Object{namespaceConfig("bar")},
			want: []runtime.Object{
				namespaceConfig("foo", markedForDeletion),
				namespaceConfig("bar"),
			},
		},
		{
			testName: "create two Namespace nodes",
			declared: []runtime.Object{
				namespaceConfig("foo"),
				namespaceConfig("bar"),
			},
			want: []runtime.Object{
				namespaceConfig("foo"),
				namespaceConfig("bar"),
			},
		},
		{
			testName: "keep one, create two, delete two Namespace nodes",
			actual: []runtime.Object{
				namespaceConfig("alp"),
				namespaceConfig("foo"),
				namespaceConfig("bar"),
			},
			declared: []runtime.Object{
				namespaceConfig("alp"),
				namespaceConfig("qux"),
				namespaceConfig("pim"),
			},
			want: []runtime.Object{
				namespaceConfig("alp"),
				namespaceConfig("foo", markedForDeletion),
				namespaceConfig("bar", markedForDeletion),
				namespaceConfig("qux"),
				namespaceConfig("pim"),
			},
		},
		// ClusterConfig tests
		{
			testName: "create ClusterConfig",
			declared: []runtime.Object{fake.ClusterConfigObject()},
			want:     []runtime.Object{fake.ClusterConfigObject()},
		},
		{
			testName: "no-op ClusterConfig",
			actual:   []runtime.Object{fake.ClusterConfigObject()},
			declared: []runtime.Object{fake.ClusterConfigObject()},
			want:     []runtime.Object{fake.ClusterConfigObject()},
		},
		{
			testName: "delete ClusterConfig",
			actual:   []runtime.Object{fake.ClusterConfigObject()},
		},
		{
			testName: "create CRD ClusterConfig",
			declared: []runtime.Object{fake.CRDClusterConfigObject()},
			want:     []runtime.Object{fake.CRDClusterConfigObject()},
		},
		{
			testName: "no-op CRD ClusterConfig",
			actual:   []runtime.Object{fake.CRDClusterConfigObject()},
			declared: []runtime.Object{fake.CRDClusterConfigObject()},
			want:     []runtime.Object{fake.CRDClusterConfigObject()},
		},
		{
			testName: "delete CRD ClusterConfig",
			actual:   []runtime.Object{fake.CRDClusterConfigObject()},
		},
		// Sync tests
		{
			testName: "create Sync",
			declared: []runtime.Object{fake.SyncObject(kinds.Anvil().GroupKind())},
			want:     []runtime.Object{fake.SyncObject(kinds.Anvil().GroupKind())},
		},
		{
			testName: "no-op Sync",
			actual:   []runtime.Object{fake.SyncObject(kinds.Anvil().GroupKind())},
			declared: []runtime.Object{fake.SyncObject(kinds.Anvil().GroupKind())},
			want:     []runtime.Object{fake.SyncObject(kinds.Anvil().GroupKind())},
		},
		{
			testName: "delete Sync",
			actual:   []runtime.Object{fake.SyncObject(kinds.Anvil().GroupKind())},
		},
		// Test all at once
		{
			testName: "multiple diffs at once",
			actual: []runtime.Object{
				fake.CRDClusterConfigObject(),
				namespaceConfig("foo"),
				namespaceConfig("bar"),
			},
			declared: []runtime.Object{
				fake.ClusterConfigObject(),
				namespaceConfig("foo"),
				namespaceConfig("qux"),
			},
			want: []runtime.Object{
				fake.ClusterConfigObject(),
				namespaceConfig("foo"),
				namespaceConfig("bar", markedForDeletion),
				namespaceConfig("qux"),
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.testName, func(t *testing.T) {
			fakeClient := testingfake.NewClient(t, runtime.NewScheme(), tc.actual...)

			actual := allConfigs(t, tc.actual)

			declared := allConfigs(t, tc.declared)

			err := Update(context.Background(), client.New(fakeClient, metrics.APICallDuration),
				testingfake.NewDecoder(nil), *actual, *declared)

			if err != nil {
				t.Errorf("unexpected error in diff: %v", err)
			}
			fakeClient.Check(t, tc.want...)
		})
	}
}
