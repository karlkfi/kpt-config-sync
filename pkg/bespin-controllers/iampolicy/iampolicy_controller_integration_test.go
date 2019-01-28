// +build integration,nonhermetic

// Warning: only run these tests using an individual's test project or the CNRM
// CI setup. Tests in this package alter the IAM policies of projects. This can
// lock out all but the owner of the organization if the tests remove some
// permission that is relied upon by others.
// TODO(cflewis): This test should also test attaching IAM policies to folders
// and organizations to get strong coverage for Bespin.
package iampolicy

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"sort"
	"strings"
	"testing"
	"time"

	bespinv1 "github.com/google/nomos/pkg/api/policyascode/v1"
	"github.com/google/nomos/pkg/bespin-controllers/terraform"
	"github.com/google/nomos/pkg/bespin-controllers/test"
	"github.com/google/nomos/pkg/bespin-controllers/test/gcp"
	"github.com/google/nomos/pkg/bespin-controllers/test/k8s"

	"github.com/google/go-cmp/cmp"
	cloudres "google.golang.org/api/cloudresourcemanager/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const resourceKind = "IAMPolicy"

var (
	cfg             *rest.Config
	executorCreator = terraform.NewTFExecutorCreator("terraform", "")
)

// TestReconcileCreateAndUpdate runs a creation and an update. The update
// is dependent on the create succeeding, so they need to be tested at the
// same time.
func TestReconcileCreateAndUpdate(t *testing.T) {
	currentUser, err := user.Current()
	if err != nil {
		t.Fatalf("unable to find current user: %v", err)
	}
	hostname, err := os.Hostname()
	if err != nil {
		t.Fatalf("unable to get hostname: %v", err)
	}

	projID := test.GetDefaultProjectID(t)

	mgr, stop := test.StartTestManager(t, cfg)
	defer stop()

	// Create the K8S project resource so the policy can attach to something.
	// Policies cannot exist without an attached resource.
	proj := &bespinv1.Project{
		TypeMeta: metav1.TypeMeta{
			Kind: bespinv1.ProjectKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      projID,
			Namespace: projID,
		},
		Spec: bespinv1.ProjectSpec{
			DisplayName: projID,
			ID:          projID,
		},
	}

	t.Logf("Creating project %v into cluster", proj.Name)
	if err := mgr.GetClient().Create(context.TODO(), proj); err != nil {
		t.Fatalf("unable to enter project into cluster: %v", err)
	}

	var tests = []struct {
		name                  string
		bindings, newBindings []bespinv1.IAMPolicyBinding
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
		{
			name: "Projects should be able to have IAM policies updated",
			bindings: []bespinv1.IAMPolicyBinding{
				{
					Role:    "roles/viewer",
					Members: []string{"group:bespin-core@google.com"},
				},
			},
			newBindings: []bespinv1.IAMPolicyBinding{
				{
					Role:    "roles/browser",
					Members: []string{"group:bespin-eng@google.com"},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Add the current service account and Googler (if run locally) as owners of the project
			// to try and prevent locking the current user out of the project.
			members := []string{fmt.Sprintf("serviceAccount:%v", test.GetDefaultServiceAccount(t))}
			if strings.HasSuffix(hostname, ".corp.google.com") {
				members = append(members, fmt.Sprintf("user:%s@google.com", currentUser.Username))
			}

			safetyBinding := bespinv1.IAMPolicyBinding{
				Role:    "roles/owner",
				Members: members,
			}

			c, err := gcp.NewCloudResourceManagerClient(context.TODO())
			if err != nil {
				t.Fatalf("unable to spawn cloudresourcemanager client: %v", err)
			}
			_, err = c.Projects.SetIamPolicy(projID, &cloudres.SetIamPolicyRequest{
				Policy: &cloudres.Policy{
					Bindings: []*cloudres.Binding{
						{
							Role:    safetyBinding.Role,
							Members: safetyBinding.Members,
						},
					},
				},
			}).Do()
			if err != nil {
				t.Fatalf("unable to set GCP safety policy: %v", err)
			}

			tc.bindings = append(tc.bindings, safetyBinding)
			tc.newBindings = append(tc.newBindings, safetyBinding)

			policy := newPolicyFixture(t, projID, tc.bindings)
			testReconcileCreate(t, mgr, projID, policy)
			if tc.newBindings != nil {
				testReconcileUpdate(t, mgr, projID, policy, tc.newBindings)
			}
		})
	}
}

func testReconcileCreate(t *testing.T, mgr manager.Manager, projID string, policy *bespinv1.IAMPolicy) {
	name := types.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}
	c := mgr.GetClient()

	t.Logf("Creating policy %v into cluster", policy.Name)
	if err := c.Create(context.TODO(), policy); err != nil {
		t.Fatalf("unable to enter policy into cluster: %v", err)
	}

	// It takes time for a fresh resource to be retrievable from the API server.
	// Retry a bit to see if it eventually arrives. Do not remove this code
	// or you will get a racy, flaky test that only creates sadness, not resources.
	if err := wait.ExponentialBackoff(wait.Backoff{Steps: 5, Duration: 250 * time.Millisecond, Factor: 1.5}, func() (bool, error) {
		if _, err := getPolicy(t, mgr, name); err != nil {
			return false, nil
		}
		return true, nil
	}); err != nil {
		t.Fatalf("policy never made it to K8S")
	}

	// Check that the policy's reconcile() was called.
	test.RunReconcilerAssertResults(t, newReconciler(mgr, &executorCreator), policy.ObjectMeta, reconcile.Result{}, nil)

	checkGCP(t, projID, policy)

	// Check that the policy resource status was updated.
	policy, err := getPolicy(t, mgr, name)
	if err != nil {
		t.Fatalf("unable to get policy: %v", err)
	}
	test.AssertReadyCondition(t, policy.Status.Conditions)
	test.AssertEventRecorded(t, &c, resourceKind, &policy.ObjectMeta, k8s.Updated)
}

func testReconcileUpdate(t *testing.T, mgr manager.Manager, projID string, policy *bespinv1.IAMPolicy, newBindings []bespinv1.IAMPolicyBinding) {
	name := types.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}
	policy, err := getPolicy(t, mgr, name)
	if err != nil {
		t.Fatalf("unable to get policy: %v", err)
	}
	c := mgr.GetClient()

	if len(newBindings) > 0 {
		policy.Spec.Bindings = newBindings
	}
	if err := c.Update(context.TODO(), policy); err != nil {
		t.Errorf("unexpected error updating k8s policy: %v", err)
	}

	// Get the policy again as the update will have updated the resource version.
	policy, err = getPolicy(t, mgr, name)
	if err != nil {
		t.Fatalf("unable to get policy: %v", err)
	}
	test.RunReconcilerAssertResults(t, newReconciler(mgr, &executorCreator), policy.ObjectMeta, reconcile.Result{}, nil)
	test.AssertEventRecorded(t, &c, resourceKind, &policy.ObjectMeta, k8s.Updated)

	checkGCP(t, projID, policy)
}

func newPolicyFixture(t *testing.T, projID string, bindings []bespinv1.IAMPolicyBinding) *bespinv1.IAMPolicy {
	t.Helper()
	if !strings.HasPrefix(t.Name(), "TestReconcile") {
		t.Fatalf("Unexpected test name prefix, all tests are expected to start with TestReconcile")
	}

	return &bespinv1.IAMPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("policy-%v", uuid.NewUUID()),
			Namespace: projID,
		},
		Spec: bespinv1.IAMPolicySpec{
			ResourceRef: corev1.ObjectReference{
				Kind: bespinv1.ProjectKind,
				Name: projID,
			},
			Bindings: bindings,
		},
	}
}

func checkGCP(t *testing.T, projID string, policy *bespinv1.IAMPolicy) {
	t.Helper()
	c, err := gcp.NewCloudResourceManagerClient(context.TODO())
	if err != nil {
		t.Fatalf("unable to spawn cloudresourcemanager client: %v", err)
	}
	gcpPolicy, err := c.Projects.GetIamPolicy(projID, &cloudres.GetIamPolicyRequest{}).Do()
	if err != nil {
		t.Fatalf("unable to get GCP policy: %v", err)
	}
	compareBindings(t, policy.Spec.Bindings, gcpPolicy.Bindings)
}

func compareBindings(t *testing.T, k8s []bespinv1.IAMPolicyBinding, gcp []*cloudres.Binding) {
	t.Helper()
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

// getPolicy gets a policy from the K8S API server.
func getPolicy(t *testing.T, mgr manager.Manager, name types.NamespacedName) (*bespinv1.IAMPolicy, error) {
	t.Helper()
	c := mgr.GetClient()
	policy := &bespinv1.IAMPolicy{}
	test.SyncCache(t, mgr)
	if err := c.Get(context.TODO(), name, policy); err != nil {
		return nil, err
	}

	return policy, nil
}

func TestMain(m *testing.M) {
	test.TestMain(m, &cfg)
}
