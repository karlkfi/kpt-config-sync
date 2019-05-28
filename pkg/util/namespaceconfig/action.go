package namespaceconfig

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	typedv1 "github.com/google/nomos/clientgen/apis/typed/configmanagement/v1"
	listersv1 "github.com/google/nomos/clientgen/listers/configmanagement/v1"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/client/action"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
)

// NewActionSpec returns a ReflectiveActionSpec appropriate for NamespaceConfig objects.
func NewActionSpec(client typedv1.ConfigmanagementV1Interface, lister listersv1.NamespaceConfigLister) *action.ReflectiveActionSpec {
	return action.NewSpec(
		new(v1.NamespaceConfig),
		v1.SchemeGroupVersion,
		NamespaceConfigsEqual,
		client,
		lister)
}

func resourceQuantityCmp(lhs, rhs resource.Quantity) bool {
	return lhs.Cmp(rhs) == 0
}

var pnsIgnore = []cmp.Option{
	// Quantity has a few unexported fields which we need to manually compare. The path is:
	// NamespaceConfigSpec -> ResourceQuota -> ResourceQuotaSpec -> ResourceList -> Quantity
	cmp.Comparer(resourceQuantityCmp),
	cmpopts.IgnoreFields(v1.NamespaceConfigSpec{}, "Token", "ImportTime"),
}

// NamespaceConfigsEqual returns true if the namespaceconfigs are equivalent.
func NamespaceConfigsEqual(lhs runtime.Object, rhs runtime.Object) bool {
	l := lhs.(*v1.NamespaceConfig)
	r := rhs.(*v1.NamespaceConfig)
	return cmp.Equal(l.Spec, r.Spec, pnsIgnore...) && action.ObjectMetaEqual(l, r)
}
