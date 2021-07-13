package reconcile

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/metadata"
	"github.com/google/nomos/pkg/metrics"
	"github.com/google/nomos/pkg/remediator/queue"
	"github.com/google/nomos/pkg/status"
	syncerclient "github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/syncertest"
	syncertestfake "github.com/google/nomos/pkg/syncer/syncertest/fake"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/testing/testmetrics"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	v1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestWorker_ProcessNextObject(t *testing.T) {
	testCases := []struct {
		name      string
		declared  []client.Object
		toProcess []client.Object
		want      []client.Object
	}{
		{
			name: "update actual objects",
			declared: []client.Object{
				fake.ClusterRoleBindingObject(syncertest.ManagementEnabled,
					core.Label("first", "one")),
				fake.ClusterRoleObject(syncertest.ManagementEnabled,
					core.Label("second", "two")),
			},
			toProcess: []client.Object{
				fake.ClusterRoleBindingObject(syncertest.ManagementEnabled),
				fake.ClusterRoleObject(syncertest.ManagementEnabled),
			},
			want: []client.Object{
				// TODO(b/162547054): Figure out why the reconciler is stripping away labels and annotations.
				fake.ClusterRoleBindingObject(syncertest.ManagementEnabled,
					core.Label("first", "one")),
				fake.ClusterRoleObject(syncertest.ManagementEnabled,
					core.Label("second", "two")),
			},
		},
		{
			name:     "delete undeclared objects",
			declared: []client.Object{},
			toProcess: []client.Object{
				fake.ClusterRoleBindingObject(syncertest.ManagementEnabled,
					core.Annotation(metadata.ResourceIDKey, "rbac.authorization.k8s.io_clusterrolebinding_default-name")),
				fake.ClusterRoleObject(syncertest.ManagementEnabled,
					core.Annotation(metadata.ResourceIDKey, "rbac.authorization.k8s.io_clusterrole_default-name")),
			},
			want: []client.Object{},
		},
		{
			name: "create missing objects",
			declared: []client.Object{
				fake.ClusterRoleBindingObject(syncertest.ManagementEnabled),
				fake.ClusterRoleObject(syncertest.ManagementEnabled),
			},
			toProcess: []client.Object{
				queue.MarkDeleted(context.Background(), fake.ClusterRoleBindingObject()),
				queue.MarkDeleted(context.Background(), fake.ClusterRoleObject()),
			},
			want: []client.Object{
				fake.ClusterRoleBindingObject(syncertest.ManagementEnabled),
				fake.ClusterRoleObject(syncertest.ManagementEnabled),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			q := queue.New("test")
			for _, obj := range tc.toProcess {
				q.Add(obj)
			}

			c := fakeClient(t)
			for _, obj := range tc.toProcess {
				if !queue.WasDeleted(context.Background(), obj) {
					if err := c.Create(context.Background(), obj); err != nil {
						t.Fatalf("Failed to create object in fake client: %v", err)
					}
				}
			}

			d := makeDeclared(t, tc.declared...)
			w := NewWorker(declared.RootReconciler, c.Applier(), q, d)

			for _, obj := range tc.toProcess {
				if ok := w.processNextObject(context.Background()); !ok {
					t.Errorf("unexpected false result from processNextObject() for object: %v", obj)
				}
			}

			c.Check(t, tc.want...)
		})
	}
}

func TestWorker_Refresh(t *testing.T) {
	name := "admin"
	namespace := "shipping"
	scheme := runtime.NewScheme()
	err := v1.AddToScheme(scheme)
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name        string
		queue       fakeQueue
		client      client.Client
		want        *unstructured.Unstructured
		wantDeleted bool
		wantErr     status.Error
	}{
		{
			name: "Not found marks object deleted",
			queue: fakeQueue{
				element: fake.UnstructuredObject(kinds.Role(), core.Name(name), core.Namespace(namespace)),
			},
			client:      syncertestfake.NewClient(t, scheme),
			want:        fake.UnstructuredObject(kinds.Role(), core.Name(name), core.Namespace(namespace)),
			wantDeleted: true,
			wantErr:     nil,
		},
		{
			name: "Found updates objects",
			queue: fakeQueue{
				element: fake.UnstructuredObject(kinds.Role(), core.Name(name), core.Namespace(namespace),
					core.Annotation("foo", "bar")),
			},
			client: syncertestfake.NewClient(t, scheme,
				fake.RoleObject(core.Name(name), core.Namespace(namespace),
					core.Annotation("foo", "qux"))),
			want: fake.UnstructuredObject(kinds.Role(), core.Name(name), core.Namespace(namespace),
				core.Annotation("foo", "qux")),
			wantDeleted: false,
			wantErr:     nil,
		},
		{
			name: "API Error does not update object",
			queue: fakeQueue{
				element: fake.UnstructuredObject(kinds.Role(), core.Name(name), core.Namespace(namespace)),
			},
			client:      syncertestfake.NewErrorClient(errors.New("some error")),
			want:        fake.UnstructuredObject(kinds.Role(), core.Name(name), core.Namespace(namespace)),
			wantDeleted: false,
			wantErr:     status.APIServerError(errors.New("some error"), ""),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := &Worker{
				objectQueue: &tc.queue,
				reconciler: fakeReconciler{
					client: tc.client,
				},
			}

			err := w.refresh(context.Background(), fake.UnstructuredObject(
				kinds.Role(), core.Name(name), core.Namespace(namespace)))
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got refresh = %v, want %v",
					err, tc.wantErr)
			}

			if !tc.wantDeleted && tc.wantErr == nil {
				// These fields are added by unstructured conversions, but we aren't
				// testing this behavior.
				_ = unstructured.SetNestedField(tc.want.Object, nil, "metadata", "creationTimestamp")
				_ = unstructured.SetNestedField(tc.want.Object, nil, "rules")
				unstructured.RemoveNestedField(tc.want.Object, "metadata", "labels")
			}

			var want client.Object = tc.want
			if tc.wantDeleted {
				want = queue.MarkDeleted(context.Background(), want)
			}

			if diff := cmp.Diff(want, tc.queue.element); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestWorker_ResourceConflictMetricValidation(t *testing.T) {
	testCases := []struct {
		name        string
		objects     []client.Object
		wantMetrics []*view.Row
	}{
		{
			name:    "single resource conflict for Role object",
			objects: []client.Object{fake.UnstructuredObject(kinds.Role())},
			wantMetrics: []*view.Row{
				{Data: &view.CountData{Value: 1}, Tags: []tag.Tag{{Key: metrics.KeyType, Value: fake.UnstructuredObject(kinds.Role()).GetKind()}}},
			},
		},
		{
			name:    "multiple resource conflicts for Role object",
			objects: []client.Object{fake.UnstructuredObject(kinds.Role()), fake.UnstructuredObject(kinds.Role())},
			wantMetrics: []*view.Row{
				{Data: &view.CountData{Value: 2}, Tags: []tag.Tag{{Key: metrics.KeyType, Value: fake.UnstructuredObject(kinds.Role()).GetKind()}}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := testmetrics.RegisterMetrics(metrics.ResourceConflictsView)

			for _, obj := range tc.objects {
				w := &Worker{
					objectQueue: &fakeQueue{},
					reconciler: fakeReconciler{
						client:       fakeClient(t),
						remediateErr: syncerclient.ConflictUpdateDoesNotExist(errors.New("resource conflict error"), obj),
					},
				}
				w.process(context.Background(), obj)
			}
			if diff := m.ValidateMetrics(metrics.ResourceConflictsView, tc.wantMetrics); diff != "" {
				t.Errorf(diff)
			}
		})
	}
}

type fakeReconciler struct {
	client       client.Client
	remediateErr status.Error
}

var _ reconcilerInterface = fakeReconciler{}

func (f fakeReconciler) Remediate(_ context.Context, _ core.ID, _ client.Object) status.Error {
	return f.remediateErr
}

func (f fakeReconciler) GetClient() client.Client {
	return f.client
}

type fakeQueue struct {
	queue.Interface
	element client.Object
}

func (q *fakeQueue) Add(o client.Object) {
	q.element = o
}

func (q *fakeQueue) Retry(o client.Object) {
	q.element = o
}

func (q *fakeQueue) Forget(_ client.Object) {
	q.element = nil
}
