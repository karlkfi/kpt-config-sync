package reconcile

import (
	"context"
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/parse/declaredresources"
	"github.com/google/nomos/pkg/policycontroller"
	syncertesting "github.com/google/nomos/pkg/syncer/testing"
	testingfake "github.com/google/nomos/pkg/syncer/testing/fake"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestRemediator_Reconcile(t *testing.T) {
	testCases := []struct {
		name string
		// version is Version (from GVK) of the object to try to remediate.
		version string
		// declared is the state of the object as returned by the Parser.
		declared core.Object
		// actual is the current state of the object on the cluster.
		actual core.Object
		// want is the desired final state of the object on the cluster after
		// reconciliation.
		want core.Object
		// wantError is the desired error resulting from calling Reconcile, if there
		// is one.
		wantError error
	}{
		// Happy Paths.
		{
			name:      "create added object",
			version:   "v1",
			declared:  fake.ClusterRoleBindingObject(),
			actual:    nil,
			want:      fake.ClusterRoleBindingObject(),
			wantError: nil,
		},
		{
			name:      "update declared object",
			version:   "v1",
			declared:  fake.ClusterRoleBindingObject(core.Label("new-label", "one")),
			actual:    fake.ClusterRoleBindingObject(),
			want:      fake.ClusterRoleBindingObject(core.Label("new-label", "one")),
			wantError: nil,
		},
		{
			name:      "delete removed object",
			version:   "v1",
			declared:  nil,
			actual:    fake.ClusterRoleBindingObject(syncertesting.ManagementEnabled),
			want:      nil,
			wantError: nil,
		},
		// Unmanaged paths.
		{
			name:      "don't create unmanaged object",
			version:   "v1",
			declared:  fake.ClusterRoleBindingObject(core.Label("declared-label", "foo"), syncertesting.ManagementDisabled),
			actual:    nil,
			want:      nil,
			wantError: nil,
		},
		{
			name:      "don't update unmanaged object",
			version:   "v1",
			declared:  fake.ClusterRoleBindingObject(core.Label("declared-label", "foo"), syncertesting.ManagementDisabled),
			actual:    fake.ClusterRoleBindingObject(core.Label("actual-label", "bar")),
			want:      fake.ClusterRoleBindingObject(core.Label("actual-label", "bar")),
			wantError: nil,
		},
		{
			name:      "don't delete unmanaged object",
			version:   "v1",
			declared:  nil,
			actual:    fake.ClusterRoleBindingObject(),
			want:      fake.ClusterRoleBindingObject(),
			wantError: nil,
		},
		// Bad declared management annotation paths.
		{
			name:      "don't create, and error on bad declared management annotation",
			version:   "v1",
			declared:  fake.ClusterRoleBindingObject(core.Label("declared-label", "foo"), syncertesting.ManagementInvalid),
			actual:    nil,
			want:      nil,
			wantError: nonhierarchical.IllegalManagementAnnotationError(fake.Namespace("namespaces/foo"), ""),
		},
		{
			name:      "don't update, and error on bad declared management annotation",
			version:   "v1",
			declared:  fake.ClusterRoleBindingObject(core.Label("declared-label", "foo"), syncertesting.ManagementInvalid),
			actual:    fake.ClusterRoleBindingObject(core.Label("actual-label", "bar")),
			want:      fake.ClusterRoleBindingObject(core.Label("actual-label", "bar")),
			wantError: nonhierarchical.IllegalManagementAnnotationError(fake.Namespace("namespaces/foo"), ""),
		},
		// bad in-cluster management annotation paths.
		{
			name:      "remove bad actual management annotation",
			version:   "v1",
			declared:  fake.ClusterRoleBindingObject(core.Label("declared-label", "foo")),
			actual:    fake.ClusterRoleBindingObject(core.Label("declared-label", "foo"), syncertesting.ManagementInvalid),
			want:      fake.ClusterRoleBindingObject(core.Label("declared-label", "foo")),
			wantError: nil,
		},
		{
			name:      "don't delete, and remove bad actual management annotation",
			version:   "v1",
			declared:  nil,
			actual:    fake.ClusterRoleBindingObject(core.Label("declared-label", "foo"), syncertesting.ManagementInvalid),
			want:      fake.ClusterRoleBindingObject(core.Label("declared-label", "foo")),
			wantError: nil,
		},
		// system namespaces
		{
			name:     "don't delete kube-system Namespace",
			version:  "v1",
			declared: nil,
			actual:   fake.NamespaceObject(metav1.NamespaceSystem, syncertesting.ManagementEnabled),
			want:     fake.NamespaceObject(metav1.NamespaceSystem),
		},
		{
			name:     "don't delete kube-public Namespace",
			version:  "v1",
			declared: nil,
			actual:   fake.NamespaceObject(metav1.NamespacePublic, syncertesting.ManagementEnabled),
			want:     fake.NamespaceObject(metav1.NamespacePublic),
		},
		{
			name:     "don't delete default Namespace",
			version:  "v1",
			declared: nil,
			actual:   fake.NamespaceObject(metav1.NamespaceDefault, syncertesting.ManagementEnabled),
			want:     fake.NamespaceObject(metav1.NamespaceDefault),
		},
		{
			name:     "don't delete gatekeeper-system Namespace",
			version:  "v1",
			declared: nil,
			actual:   fake.NamespaceObject(policycontroller.NamespaceSystem, syncertesting.ManagementEnabled),
			want:     fake.NamespaceObject(policycontroller.NamespaceSystem),
		},
		// Version difference paths.
		{
			name:      "update actual object with different version",
			declared:  fake.ClusterRoleBindingV1Beta1Object(core.Label("new-label", "one")),
			actual:    fake.ClusterRoleBindingObject(),
			want:      fake.ClusterRoleBindingV1Beta1Object(core.Label("new-label", "one")),
			wantError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up the fake client that represents the initial state of the cluster.
			c := fakeClient(t, tc.actual)
			// Simulate the Parser having already parsed the resource and recorded it.
			d := declared(t, tc.declared)

			r := newReconciler(c, c.Applier(), d)

			// Get the triggering object for the reconcile event.
			var obj core.Object
			switch {
			case tc.declared != nil:
				obj = tc.declared
			case tc.actual != nil:
				obj = tc.actual
			default:
				t.Fatal("at least one of actual or declared must be specified for a test")
			}

			err := r.Remediate(context.Background(), core.IDOf(obj), tc.actual)
			if !errors.Is(err, tc.wantError) {
				t.Errorf("got Reconcile() = %v, want matching %v",
					err, tc.wantError)
			}

			if tc.want == nil {
				c.Check(t)
			} else {
				c.Check(t, tc.want)
			}
		})
	}
}

func fakeClient(t *testing.T, actual core.Object) *testingfake.Client {
	t.Helper()
	s := runtime.NewScheme()
	err := corev1.AddToScheme(s)
	if err != nil {
		t.Fatal(err)
	}
	err = rbacv1.AddToScheme(s)
	if err != nil {
		t.Fatal(err)
	}

	err = rbacv1beta1.AddToScheme(s)
	if err != nil {
		t.Fatal(err)
	}

	c := testingfake.NewClient(t, s)
	if actual != nil {
		err := c.Create(context.Background(), actual)
		if err != nil {
			// Test precondition; fail early.
			t.Fatal(err)
		}
	}
	return c
}

func declared(t *testing.T, declared core.Object) *declaredresources.DeclaredResources {
	t.Helper()
	d := declaredresources.NewDeclaredResources()
	if declared != nil {
		err := d.UpdateDecls([]core.Object{declared})
		if err != nil {
			// Test precondition; fail early.
			t.Fatal(err)
		}
	}
	return d
}
