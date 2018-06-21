package clusterpolicy

import (
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/apis/core/validation"
)

// Validate returns an error if the ClusterPolicy has an invalid name or
// contains sub-resources with duplicate names.
func Validate(clusterPolicy *policyhierarchy_v1.ClusterPolicy) error {
	if clusterPolicy.Name != policyhierarchy_v1.ClusterPolicyName {
		return errors.Errorf("invalid name %q, should be %q", clusterPolicy.Name, policyhierarchy_v1.ClusterPolicyName)
	}
	clusterRoleNames := make(map[string]bool)
	for _, r := range clusterPolicy.Spec.ClusterRolesV1 {
		if err := validateNameLen(&r); err != nil {
			return err
		}
		n := r.Name
		if clusterRoleNames[n] {
			return errors.Errorf("duplicate clusterrole name %q in clusterpolicy", n)
		}
		clusterRoleNames[n] = true
	}
	clusterRoleBindingNames := make(map[string]bool)
	for _, rb := range clusterPolicy.Spec.ClusterRoleBindingsV1 {
		if err := validateNameLen(&rb); err != nil {
			return err
		}
		n := rb.Name
		if clusterRoleBindingNames[n] {
			return errors.Errorf("duplicate clusterrolebinding name %q in clusterpolicy", n)
		}
		clusterRoleBindingNames[n] = true
	}
	podSecurityPolicyNames := make(map[string]bool)
	for _, psp := range clusterPolicy.Spec.PodSecurityPoliciesV1Beta1 {
		if err := validateNameLen(&psp); err != nil {
			return err
		}
		n := psp.Name
		if podSecurityPolicyNames[n] {
			return errors.Errorf("duplicate podsecuritypolicy name %q in clusterpolicy", n)
		}
		podSecurityPolicyNames[n] = true
	}

	return nil
}

func validateNameLen(obj metav1.Object) error {
	if !validation.IsValidSysctlName(obj.GetName()) {
		return errors.Errorf(
			"resource has invalid name: %s %s",
			obj.(runtime.Object).GetObjectKind().GroupVersionKind(),
			obj.GetName(),
		)
	}
	return nil
}
