package namespaceconfig

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/syncer/decode"
	"github.com/google/nomos/pkg/util/compare"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
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
func NamespaceConfigsEqual(decoder decode.Decoder, lhs runtime.Object, rhs runtime.Object) (bool, error) {
	l := lhs.(*v1.NamespaceConfig)
	r := rhs.(*v1.NamespaceConfig)

	resourceEqual, err := compare.GenericResourcesEqual(decoder, l.Spec.Resources, r.Spec.Resources, ncsIgnore...)
	if err != nil {
		return false, err
	}
	metaEqual := compare.ObjectMetaEqual(l, r)
	// We only care about the DeleteSyncedTime field in .spec; all the other fields are
	// expected to change in between reconciles.
	// TODO(b/135766013): fix e2e tests to expect namespaceconfigs to not always be updated every reconcile and then remove
	//  the .spec.Token check.
	namespaceConfigsEqual := resourceEqual && metaEqual && l.Spec.DeleteSyncedTime == r.Spec.DeleteSyncedTime && l.Spec.Token == r.Spec.Token
	return namespaceConfigsEqual, nil
}
