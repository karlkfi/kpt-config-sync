package policynode

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	typedv1 "github.com/google/nomos/clientgen/apis/typed/policyhierarchy/v1"
	listersv1 "github.com/google/nomos/clientgen/listers/policyhierarchy/v1"
	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/client/action"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
)

// NewActionSpec returns a ReflectiveActionSpec appropriate for PolicyNode objects.
func NewActionSpec(client typedv1.NomosV1Interface, lister listersv1.PolicyNodeLister) *action.ReflectiveActionSpec {
	return action.NewSpec(
		new(v1.PolicyNode),
		v1.SchemeGroupVersion,
		policyNodesEqual,
		client,
		lister)
}

func resourceQuantityCmp(lhs, rhs resource.Quantity) bool {
	return lhs.Cmp(rhs) == 0
}

var pnsIgnore = []cmp.Option{
	// Quantity has a few unexported fields which we need to manually compare. The path is:
	// PolicyNodeSpec -> ResourceQuota -> ResourceQuotaSpec -> ResourceList -> Quantity
	cmp.Comparer(resourceQuantityCmp),
	cmpopts.IgnoreFields(v1.PolicyNodeSpec{}, "ImportToken", "ImportTime"),
}

func policyNodesEqual(lhs runtime.Object, rhs runtime.Object) bool {
	l := lhs.(*v1.PolicyNode)
	r := rhs.(*v1.PolicyNode)
	return cmp.Equal(l.Spec, r.Spec, pnsIgnore...)
}
