package namespaceconfig

import (
	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/syncer/decode"
	"github.com/google/nomos/pkg/util/compare"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func resourceQuantityCmp(lhs, rhs resource.Quantity) bool {
	return lhs.Cmp(rhs) == 0
}

var ncsIgnore = []cmp.Option{
	// Quantity has a few unexported fields which we need to manually compare. The path is:
	// ResourceQuota -> ResourceQuotaSpec -> ResourceList -> Quantity
	cmp.Comparer(resourceQuantityCmp),
}

// NamespaceConfigsEqual returns true if the NamespaceConfigs are equivalent.
func NamespaceConfigsEqual(decoder decode.Decoder, lhs client.Object, rhs client.Object) (bool, error) {
	l := lhs.(*v1.NamespaceConfig)
	r := rhs.(*v1.NamespaceConfig)

	resourceEqual, err := compare.GenericResourcesEqual(decoder, l.Spec.Resources, r.Spec.Resources, ncsIgnore...)
	if err != nil {
		return false, err
	}
	metaEqual := compare.ObjectMetaEqual(l, r)
	// We only care about the DeleteSyncedTime field in .spec; all the other fields are
	// expected to change in between reconciles.
	namespaceConfigsEqual := resourceEqual && metaEqual && l.Spec.DeleteSyncedTime == r.Spec.DeleteSyncedTime
	return namespaceConfigsEqual, nil
}
