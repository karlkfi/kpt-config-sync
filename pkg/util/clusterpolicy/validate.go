package clusterpolicy

import (
	policyhierarchyv1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/util/meta"
	"github.com/pkg/errors"
)

// Validate returns an error if the ClusterPolicy has an invalid name or
// contains sub-resources with duplicate names.
func Validate(clusterPolicy *policyhierarchyv1.ClusterPolicy) error {
	if clusterPolicy.Name != policyhierarchyv1.ClusterPolicyName {
		return errors.Errorf("invalid ClusterPolicy name %q, should be %q", clusterPolicy.Name, policyhierarchyv1.ClusterPolicyName)
	}
	validator := meta.NewValidator()
	for _, resList := range []interface{}{
		clusterPolicy.Spec.ClusterRolesV1,
		clusterPolicy.Spec.ClusterRoleBindingsV1,
		clusterPolicy.Spec.PodSecurityPoliciesV1Beta1,
	} {
		if err := validator.Validate(resList); err != nil {
			return err
		}
	}
	return nil
}
