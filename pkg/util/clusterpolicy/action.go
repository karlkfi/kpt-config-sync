package clusterpolicy

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	typedv1 "github.com/google/nomos/clientgen/apis/typed/policyhierarchy/v1"
	listersv1 "github.com/google/nomos/clientgen/listers/policyhierarchy/v1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/client/action"
	"k8s.io/apimachinery/pkg/runtime"
)

// NewActionSpec returns a ReflectiveActionSpec appropriate for ClusterPolicy objects.
func NewActionSpec(client typedv1.NomosV1Interface, lister listersv1.ClusterPolicyLister) *action.ReflectiveActionSpec {
	return action.NewSpec(
		new(v1.ClusterPolicy),
		v1.SchemeGroupVersion,
		clusterPoliciesEqual,
		client,
		lister)
}

var cpsIgnore = []cmp.Option{
	cmpopts.IgnoreFields(v1.ClusterPolicySpec{}, "ImportToken", "ImportTime"),
}

func clusterPoliciesEqual(lhs runtime.Object, rhs runtime.Object) bool {
	l := lhs.(*v1.ClusterPolicy)
	r := rhs.(*v1.ClusterPolicy)
	return cmp.Equal(l.Spec, r.Spec, cpsIgnore...)
}
