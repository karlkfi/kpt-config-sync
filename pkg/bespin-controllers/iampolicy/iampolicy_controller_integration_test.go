// +build integration,nonhermetic

// Warning: only run these tests using an individual's test project or the CNRM
// CI setup.
// Tests in this package alter the IAM policies of projects. This can lock out
// all but the owner of the organization if the tests remove some permission that is
// relied upon by others.
// TODO(cflewis): This test should also test attaching IAM policies to folders and organizations
// to get strong coverage for Bespin.
package iampolicy

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/user"
	"sort"
	"strings"
	"testing"

	bespinv1 "github.com/google/nomos/pkg/api/policyascode/v1"
	"github.com/google/nomos/pkg/bespin-controllers/test"
	"github.com/google/nomos/pkg/bespin-controllers/test/gcp"
	"github.com/google/nomos/pkg/bespin-controllers/test/k8s"

	"github.com/google/go-cmp/cmp"
	cloudres "google.golang.org/api/cloudresourcemanager/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const resourceKind = "IAMPolicy"

var (
	cfg *rest.Config
)

func TestReconcileCreate(t *testing.T) {
	if err := flag.Set("local", "true"); err != nil {
		t.Fatalf("unable to run Terraform locally")
	}

	currentUser, err := user.Current()
	if err != nil {
		t.Fatalf("unable to find current user: %v", err)
	}
	hostname, err := os.Hostname()
	if err != nil {
		t.Fatalf("unable to get hostname: %v", err)
	}

	// Warning: These tests will overwrite any current IAM policy. If a policy is not set
	// where you have at least editor permissions, you will lock yourself out
	// of editing the project further and the project is essentially dead.
	var tests = []struct {
		name     string
		bindings []bespinv1.IAMPolicyBinding
	}{
		{
			name: "Projects should be able to have IAM policies set",
			bindings: []bespinv1.IAMPolicyBinding{
				{
					Role:    "roles/editor",
					Members: []string{"group:kcc-eng@google.com"},
				},
			},
		},
	}

	mgr, stop := test.StartTestManager(t, cfg)
	defer stop()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Add the current service account and Googler (if run locally) as owners of the project
			// to try and prevent lockouts.
			members := []string{fmt.Sprintf("serviceAccount:%v", test.GetDefaultServiceAccount(t))}
			if strings.HasSuffix(hostname, ".corp.google.com") {
				members = append(members, fmt.Sprintf("user:%s@google.com", currentUser.Username))
			}
			tc.bindings = append(tc.bindings,
				bespinv1.IAMPolicyBinding{
					Role:    "roles/owner",
					Members: members,
				})
			projID := test.GetDefaultProjectID(t)
			policy := newPolicyFixture(t, projID, tc.bindings)

			// Create the K8S project resource so the policy can attach to something.
			// Policies cannot exist without an attached resource.
			proj := &bespinv1.Project{
				TypeMeta: metav1.TypeMeta{
					Kind: policy.Spec.ResourceRef.Kind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      policy.Spec.ResourceRef.Name,
					Namespace: projID,
				},
				Spec: bespinv1.ProjectSpec{
					DisplayName: policy.Spec.ResourceRef.Name,
					ID:          projID,
				},
			}

			// Normally integration tests would remove the current resource to ensure the
			// test is hermetic. However, removing all IAM bindings will lock the owners and the
			// service account this test runs as, out of the project.
			mgrC := mgr.GetClient()

			t.Logf("Creating project %v into cluster", proj.Name)
			if err := mgrC.Create(context.TODO(), proj); err != nil {
				t.Fatalf("unable to enter project into cluster: %v", err)
			}

			// Add the policy to the K8S cluster.
			t.Logf("Creating policy %v into cluster", policy.Name)
			if err := mgrC.Create(context.TODO(), policy); err != nil {
				t.Fatalf("unable to enter policy into cluster: %v", err)
			}

			// Check that the policy's reconcile() was called.
			test.RunReconcilerAssertResults(t, newReconciler(mgr), policy.ObjectMeta, reconcile.Result{}, nil)

			// Check that the policy was realized on GCP.
			c, err := gcp.NewCloudResourceManagerClient(context.TODO())
			if err != nil {
				t.Fatalf("unable to spawn cloudresourcemanager client: %v", err)
			}
			gcpPolicy, err := c.Projects.GetIamPolicy(proj.Name, &cloudres.GetIamPolicyRequest{}).Do()
			if err != nil {
				t.Fatalf("unable to get GCP policy: %v", err)
			}
			compareBindings(t, policy.Spec.Bindings, gcpPolicy.Bindings)

			// Check that the policy resource status was updated.
			test.SyncCache(t, mgr)
			name := types.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}
			err = mgrC.Get(context.TODO(), name, policy)
			if err != nil {
				t.Fatalf("unexpected error getting k8s policy: %v", err)
			}
			test.AssertReadyCondition(t, policy.Status.Conditions)
			test.AssertEventRecorded(t, &mgrC, resourceKind, &policy.ObjectMeta, k8s.Updated)
		})
	}
}

func newPolicyFixture(t *testing.T, projectID string, bindings []bespinv1.IAMPolicyBinding) *bespinv1.IAMPolicy {
	t.Helper()
	if projectID == "" {
		t.Fatalf("project ID must be not nil")
	}
	if !strings.HasPrefix(t.Name(), "TestReconcile") {
		t.Fatalf("Unexpected test name prefix, all tests are expected to start with TestReconcile")
	}

	return &bespinv1.IAMPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("policy-%v", uuid.NewUUID()),
			Namespace: projectID,
		},
		Spec: bespinv1.IAMPolicySpec{
			ResourceRef: corev1.ObjectReference{
				Kind: bespinv1.ProjectKind,
				Name: projectID,
			},
			Bindings: bindings,
		},
	}
}

func compareBindings(t *testing.T, k8s []bespinv1.IAMPolicyBinding, gcp []*cloudres.Binding) {
	for _, k8sBind := range k8s {
		sort.Strings(k8sBind.Members)
		var found bool
		for _, gcpBind := range gcp {
			sort.Strings(gcpBind.Members)
			t.Logf("Comparing %v / %+v and %v / %+v", k8sBind.Role, k8sBind.Members, gcpBind.Role, gcpBind.Members)
			if cmp.Equal(k8sBind.Role, gcpBind.Role) && cmp.Equal(k8sBind.Members, gcpBind.Members) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("want %+v, got %+v", k8s, gcp)
		}
	}
}

func TestMain(m *testing.M) {
	test.TestMain(m, &cfg)
}
