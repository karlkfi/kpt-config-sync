package e2e

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/e2e/nomostest"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/pkg/errors"
	rbacv1 "k8s.io/api/rbac/v1"
)

// sortPolicyRules sorts PolicyRules lexicographically by JSON representation.
func sortPolicyRules(l, r rbacv1.PolicyRule) bool {
	jsnL, _ := json.Marshal(l)
	jsnR, _ := json.Marshal(r)
	return string(jsnL) < string(jsnR)
}

func hasRules(rules []rbacv1.PolicyRule) nomostest.Predicate {
	return func(o core.Object) error {
		cr, ok := o.(*rbacv1.ClusterRole)
		if !ok {
			return nomostest.WrongTypeErr(cr, &rbacv1.ClusterRole{})
		}

		// Ignore the order of the policy rules.
		if diff := cmp.Diff(rules, cr.Rules, cmpopts.SortSlices(sortPolicyRules)); diff != "" {
			return errors.New(diff)
		}
		return nil
	}
}

// TestRevertClusterRole ensures that we revert conflicting manually-applied
// changes to cluster-scoped objects.
func TestRevertClusterRole(t *testing.T) {
	t.Parallel()
	nt := nomostest.New(t)

	crName := "e2e-test-clusterrole"

	err := nt.ValidateNotFound(crName, "", fake.ClusterRoleObject())
	if err != nil {
		t.Fatal(err)
	}

	// Declare the ClusterRole.
	declaredRules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{kinds.Deployment().Kind},
			Verbs:     []string{"get", "list", "create"},
		},
	}
	declaredcr := fake.ClusterRoleObject(core.Name(crName))
	declaredcr.Rules = declaredRules
	nt.Repository.Add("acme/cluster/clusterrole.yaml", declaredcr)
	nt.Repository.CommitAndPush("add get/list/create ClusterRole")
	nt.WaitForRepoSync()

	err = nt.Validate(crName, "", &rbacv1.ClusterRole{},
		hasRules(declaredRules))
	if err != nil {
		t.Fatalf("validating ClusterRole precondition: %v", err)
	}

	// Apply a conflicting ClusterRole.
	appliedRules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{kinds.Deployment().Kind},
			Verbs:     []string{"get", "list"}, // missing "create"
		},
	}
	appliedcr := fake.ClusterRoleObject(core.Name(crName))
	appliedcr.Rules = appliedRules
	err = nt.Update(appliedcr)
	if err != nil {
		t.Fatalf("applying conflicting ClusterRole: %v", err)
	}

	// Ensure the conflict is reverted.
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate(crName, "", &rbacv1.ClusterRole{},
			hasRules(declaredRules))
	})
	if err != nil {
		t.Errorf("reverting ClusterRole conflict: %v", err)
	}
}

// TestClusterRoleLifecycle ensures we can add/update/delete cluster-scoped
// resources.
func TestClusterRoleLifecycle(t *testing.T) {
	t.Parallel()
	nt := nomostest.New(t)

	crName := "e2e-test-clusterrole"

	err := nt.ValidateNotFound(crName, "", fake.ClusterRoleObject())
	if err != nil {
		t.Fatal(err)
	}

	// Declare the ClusterRole in repo.
	declaredRules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{kinds.Deployment().Kind},
			Verbs:     []string{"get", "list", "create"},
		},
	}
	declaredcr := fake.ClusterRoleObject(core.Name(crName))
	declaredcr.Rules = declaredRules
	nt.Repository.Add("acme/cluster/clusterrole.yaml", declaredcr)
	nt.Repository.CommitAndPush("add get/list/create ClusterRole")
	nt.WaitForRepoSync()
	// Validate ClusterConfig behavior.
	nt.WaitForSync(func() core.Object { return &v1.ClusterConfig{} }, v1.ClusterConfigName,
		nomostest.ClusterConfigHasSpecToken,
		nomostest.ClusterConfigHasStatusToken,
	)

	err = nt.Validate(crName, "", &rbacv1.ClusterRole{},
		hasRules(declaredRules))
	if err != nil {
		t.Fatalf("validating ClusterRole precondition: %v", err)
	}

	// Update the ClusterRole in the SOT.
	updatedRules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{kinds.Deployment().Kind},
			Verbs:     []string{"get", "list"}, // missing "create"
		},
	}
	updatedcr := fake.ClusterRoleObject(core.Name(crName))
	updatedcr.Rules = updatedRules
	nt.Repository.Add("acme/cluster/clusterrole.yaml", updatedcr)
	nt.Repository.CommitAndPush("update ClusterRole to get/list")

	// Ensure the resource is updated.
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate(crName, "", &rbacv1.ClusterRole{},
			hasRules(updatedRules))
	})
	if err != nil {
		t.Errorf("updating ClusterRole: %v", err)
	}

	// Delete the ClusterRole from the SOT.
	nt.Repository.Remove("acme/cluster/clusterrole.yaml")
	nt.Repository.CommitAndPush("deleting ClusterRole")
	nt.WaitForRepoSync()

	err = nt.ValidateNotFound(crName, "", &rbacv1.ClusterRole{})
	if err != nil {
		t.Errorf("deleting ClusterRole: %v", err)
	}
}
