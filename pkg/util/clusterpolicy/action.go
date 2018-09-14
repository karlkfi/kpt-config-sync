package clusterpolicy

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	typed_v1 "github.com/google/nomos/clientgen/apis/typed/nomos/v1"
	listers_v1 "github.com/google/nomos/clientgen/listers/nomos/v1"
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/nomos/v1"
	"github.com/google/nomos/pkg/client/action"
	"k8s.io/apimachinery/pkg/runtime"
)

// NewActionSpec returns a ReflectiveActionSpec appropriate for ClusterPolicy objects.
func NewActionSpec(client typed_v1.NomosV1Interface, lister listers_v1.ClusterPolicyLister) *action.ReflectiveActionSpec {
	return action.NewSpec(
		new(policyhierarchy_v1.ClusterPolicy),
		policyhierarchy_v1.SchemeGroupVersion,
		clusterPoliciesEqual,
		client,
		lister)
}

var cpsIgnore = []cmp.Option{
	cmpopts.IgnoreFields(policyhierarchy_v1.ClusterPolicySpec{}, "ImportToken", "ImportTime"),
}

func clusterPoliciesEqual(lhs runtime.Object, rhs runtime.Object) bool {
	l := lhs.(*policyhierarchy_v1.ClusterPolicy)
	r := rhs.(*policyhierarchy_v1.ClusterPolicy)
	return cmp.Equal(l.Spec, r.Spec, cpsIgnore...)
}
