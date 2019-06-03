package namespaceconfig

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/util/compare"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
)

func resourceQuantityCmp(lhs, rhs resource.Quantity) bool {
	return lhs.Cmp(rhs) == 0
}

var ncsIgnore = []cmp.Option{
	// Quantity has a few unexported fields which we need to manually compare. The path is:
	// NamespaceConfigSpec -> ResourceQuota -> ResourceQuotaSpec -> ResourceList -> Quantity
	cmp.Comparer(resourceQuantityCmp),
	cmpopts.IgnoreFields(v1.NamespaceConfigSpec{}, "Token", "ImportTime"),
}

// NamespaceConfigsEqual returns true if the NamespaceConfigs are equivalent.
func NamespaceConfigsEqual(lhs runtime.Object, rhs runtime.Object) bool {
	l := lhs.(*v1.NamespaceConfig)
	r := rhs.(*v1.NamespaceConfig)
	return cmp.Equal(l.Spec, r.Spec, ncsIgnore...) && compare.ObjectMetaEqual(l, r)
}
