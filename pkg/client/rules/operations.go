package rules

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	apipolicyhierarchy "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/pkg/errors"
	"k8s.io/client-go/tools/cache"
)

// GetPolicyRules returns all the PolicyNodeSpecs that apply to 'namespace'
// which are known to the supplied informer at the time the request is made.
// The returned PolicyNodeSpec index 0 is the policy node spec for the leaf
// namespace.  The last policy node returned is for the root namespace, and the
// namespaces in between are namespaces on the path between the two in the
// appropriate sequence.
func GetPolicyRules(i cache.SharedIndexInformer, namespace string) (
	*[]apipolicyhierarchy.PolicyNodeSpec, error) {
	// Perhaps it is OK to return any policy node spec that has been
	// built so far.
	policies := make([]apipolicyhierarchy.PolicyNodeSpec, 0)

	glog.V(5).Infof("GetPolicyRules: namespace=%v", namespace)

	// Follows a trail of namespaces starting from 'namespace', then
	// following the back-pointers to parents, up to the root PolicyNode.
	var err error
	nextNamespace := namespace
	for nextNamespace != apipolicyhierarchy.NoParentNamespace {
		glog.V(6).Infof("policyRulesFor: resolving namespace: %v", nextNamespace)
		// TODO(fmil): Use a typed informer instead.
		rawPolicyNode, exists, loopErr := i.GetStore().
			GetByKey(nextNamespace)
		if loopErr != nil {
			err = errors.Wrapf(
				loopErr, "while resolving namespace: %v",
				nextNamespace)
			break
		}
		if !exists {
			err = errors.Errorf("partial policy rules, missing namespace: %v",
				nextNamespace)
			break
		}
		policyNode := rawPolicyNode.(*apipolicyhierarchy.PolicyNode)
		policyNodeSpec := policyNode.Spec
		policies = append(policies, policyNodeSpec)
		nextNamespace = policyNodeSpec.Parent
	}
	glog.V(5).Infof("policyRulesFor: policies=%v, err=%v",
		spew.Sdump(policies), err)
	if err != nil {
		return &policies, errors.Wrapf(
			err, "while getting policy node: %s", namespace)
	}
	return &policies, nil
}
