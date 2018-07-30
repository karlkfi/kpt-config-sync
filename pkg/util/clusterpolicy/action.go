package clusterpolicy

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	listers_v1 "github.com/google/nomos/clientgen/listers/policyhierarchy/v1"
	typed_v1 "github.com/google/nomos/clientgen/policyhierarchy/typed/policyhierarchy/v1"
	api_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/client/action"
	"k8s.io/apimachinery/pkg/runtime"
)

// NewActionSpec returns a ReflectiveActionSpec appropriate for ClusterPolicy objects.
func NewActionSpec(client typed_v1.NomosV1Interface, lister listers_v1.ClusterPolicyLister) *action.ReflectiveActionSpec {
	return &action.ReflectiveActionSpec{
		Resource:   action.LowerPlural(policyhierarchy_v1.ClusterPolicy{}),
		KindPlural: action.Plural(policyhierarchy_v1.ClusterPolicy{}),
		Group:      api_v1.GroupName,
		Version:    api_v1.SchemeGroupVersion.Version,
		EqualSpec:  clusterPoliciesEqual,
		Client:     client,
		Lister:     lister,
	}
}

var cpsIgnore = []cmp.Option{
	cmpopts.IgnoreFields(policyhierarchy_v1.ClusterPolicySpec{}, "ImportToken", "ImportTime"),
}

func clusterPoliciesEqual(lhs runtime.Object, rhs runtime.Object) bool {
	l := lhs.(*policyhierarchy_v1.ClusterPolicy)
	r := rhs.(*policyhierarchy_v1.ClusterPolicy)
	return cmp.Equal(l.Spec, r.Spec, cpsIgnore...)
}
