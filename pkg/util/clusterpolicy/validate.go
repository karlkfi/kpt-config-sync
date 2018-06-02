package clusterpolicy

import (
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/pkg/errors"
)

// Validate returns an error if the ClusterPolicy has an invalid name or
// contains sub-resources with duplicate names.
func Validate(clusterPolicy *policyhierarchy_v1.ClusterPolicy) error {
	if clusterPolicy.Name != policyhierarchy_v1.ClusterPolicyName {
		return errors.Errorf("invalid name %q, should be %q", clusterPolicy.Name, policyhierarchy_v1.ClusterPolicyName)
	}
	clusterRoleNames := make(map[string]bool)
	for _, r := range clusterPolicy.Spec.ClusterRolesV1 {
		n := r.Name
		if clusterRoleNames[n] {
			return errors.Errorf("duplicate clusterrole name %q in clusterpolicy", n)
		}
		clusterRoleNames[n] = true
	}
	clusterRoleBindingNames := make(map[string]bool)
	for _, rb := range clusterPolicy.Spec.ClusterRoleBindingsV1 {
		n := rb.Name
		if clusterRoleBindingNames[n] {
			return errors.Errorf("duplicate clusterrolebinding name %q in clusterpolicy", n)
		}
		clusterRoleBindingNames[n] = true
	}
	podSecurityPolicyNames := make(map[string]bool)
	for _, psp := range clusterPolicy.Spec.PodSecurityPoliciesV1Beta1 {
		n := psp.Name
		if podSecurityPolicyNames[n] {
			return errors.Errorf("duplicate podsecuritypolicy name %q in clusterpolicy", n)
		}
		podSecurityPolicyNames[n] = true
	}

	return nil
}
