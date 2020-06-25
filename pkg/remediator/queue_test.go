package remediator

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/core"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func gvknn(group, version, kind, namespace, name string) GVKNN {
	return GVKNN{
		ID: core.ID{
			GroupKind: schema.GroupKind{
				Group: group,
				Kind:  kind,
			},
			ObjectKey: client.ObjectKey{
				Namespace: namespace,
				Name:      name,
			},
		},
		Version: version,
	}
}

func TestQueue(t *testing.T) {
	testCases := []struct {
		name    string
		entries []GVKNN
		want    []GVKNN
	}{
		// Trivial cases
		{
			name:    "no entries returns nothing",
			entries: nil,
			want:    nil,
		},
		{
			name: "one entry returns the entry",
			entries: []GVKNN{
				gvknn(rbacv1.GroupName, "v1", "Role", "shipping", "user"),
			},
			want: []GVKNN{
				gvknn(rbacv1.GroupName, "v1", "Role", "shipping", "user"),
			},
		},
		// Single-difference cases
		{
			name: "different groups results in two entries",
			entries: []GVKNN{
				gvknn(rbacv1.GroupName, "v1", "Role", "shipping", "user"),
				gvknn(corev1.GroupName, "v1", "Role", "shipping", "user"),
			},
			want: []GVKNN{
				gvknn(rbacv1.GroupName, "v1", "Role", "shipping", "user"),
				gvknn(corev1.GroupName, "v1", "Role", "shipping", "user"),
			},
		},
		{
			name: "different versions results in two entries",
			entries: []GVKNN{
				gvknn(rbacv1.GroupName, "v1", "Role", "shipping", "user"),
				gvknn(rbacv1.GroupName, "v1beta1", "Role", "shipping", "user"),
			},
			want: []GVKNN{
				gvknn(rbacv1.GroupName, "v1", "Role", "shipping", "user"),
				gvknn(rbacv1.GroupName, "v1beta1", "Role", "shipping", "user"),
			},
		},
		{
			name: "different kinds results in two entries",
			entries: []GVKNN{
				gvknn(rbacv1.GroupName, "v1", "Role", "shipping", "user"),
				gvknn(rbacv1.GroupName, "v1", "RoleBinding", "shipping", "user"),
			},
			want: []GVKNN{
				gvknn(rbacv1.GroupName, "v1", "Role", "shipping", "user"),
				gvknn(rbacv1.GroupName, "v1", "RoleBinding", "shipping", "user"),
			},
		},
		{
			name: "different namespaces results in two entries",
			entries: []GVKNN{
				gvknn(rbacv1.GroupName, "v1", "Role", "shipping", "user"),
				gvknn(rbacv1.GroupName, "v1", "Role", "accounts", "user"),
			},
			want: []GVKNN{
				gvknn(rbacv1.GroupName, "v1", "Role", "shipping", "user"),
				gvknn(rbacv1.GroupName, "v1", "Role", "accounts", "user"),
			},
		},
		{
			name: "different names results in two entries",
			entries: []GVKNN{
				gvknn(rbacv1.GroupName, "v1", "Role", "shipping", "user"),
				gvknn(rbacv1.GroupName, "v1", "Role", "shipping", "admin"),
			},
			want: []GVKNN{
				gvknn(rbacv1.GroupName, "v1", "Role", "shipping", "user"),
				gvknn(rbacv1.GroupName, "v1", "Role", "shipping", "admin"),
			},
		},
		// Deduplication behavior.
		{
			name: "duplicate entries returns the entry",
			entries: []GVKNN{
				gvknn(rbacv1.GroupName, "v1", "Role", "shipping", "user"),
				gvknn(rbacv1.GroupName, "v1", "Role", "shipping", "user"),
			},
			want: []GVKNN{
				gvknn(rbacv1.GroupName, "v1", "Role", "shipping", "user"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			q := NewQueue()

			// We're doing sequential adds, so the resulting queue is deterministic.
			for _, e := range tc.entries {
				q.Add(e)
			}

			var got []GVKNN
			for q.Len() > 0 {
				// We normally couldn't do this, but since this test is the only thread
				// working on the queue it gives the expected results.
				id, _ := q.Get()
				got = append(got, *id)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestQueue_ReAddDuringProcessing(t *testing.T) {
	q := NewQueue()
	o := gvknn("rbac", "v1", "Role", "shipping", "admin")

	// Mark the item as dirty, then get it from the queue.
	q.Add(o)
	_, _ = q.Get()

	// Mark the item as dirty again before finishing processing it.
	q.Add(o)
	q.Done(o)

	// Ensure the item is still marked as needing processing.
	l := q.Len()
	if l != 1 {
		t.Errorf("got q.Len() = %d, want %d", l, 1)
	}
}
