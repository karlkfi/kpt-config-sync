package policynode

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	typed_v1 "github.com/google/nomos/clientgen/apis/typed/policyhierarchy/v1"
	listers_v1 "github.com/google/nomos/clientgen/listers/policyhierarchy/v1"
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/client/action"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
)

// NewActionSpec returns a ReflectiveActionSpec appropriate for PolicyNode objects.
func NewActionSpec(client typed_v1.NomosV1Interface, lister listers_v1.PolicyNodeLister) *action.ReflectiveActionSpec {
	return action.NewSpec(
		new(policyhierarchy_v1.PolicyNode),
		policyhierarchy_v1.SchemeGroupVersion,
		policyNodesEqual,
		client,
		lister)
}

var pnsIgnore = []cmp.Option{
	// Quantity has a few unexported fields which we need to explicitly ignore. The path is:
	// PolicyNodeSpec -> ResourceQuota -> ResourceQuotaSpec -> ResourceList -> Quantity
	cmpopts.IgnoreUnexported(resource.Quantity{}),
	cmpopts.IgnoreFields(policyhierarchy_v1.PolicyNodeSpec{}, "ImportToken", "ImportTime"),
}

func policyNodesEqual(lhs runtime.Object, rhs runtime.Object) bool {
	l := lhs.(*policyhierarchy_v1.PolicyNode)
	r := rhs.(*policyhierarchy_v1.PolicyNode)
	return cmp.Equal(l.Spec, r.Spec, pnsIgnore...)
}
