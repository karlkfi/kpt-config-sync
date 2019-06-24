package clusterconfig

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/syncer/decode"
	"github.com/google/nomos/pkg/util/compare"
	"k8s.io/apimachinery/pkg/runtime"
)

// ClusterConfigsEqual returns true if the clusterconfigs are equivalent.
func ClusterConfigsEqual(decoder decode.Decoder, lhs runtime.Object, rhs runtime.Object) (bool, error) {
	l := lhs.(*v1.ClusterConfig)
	r := rhs.(*v1.ClusterConfig)

	return compare.GenericResourcesEqual(decoder, l.Spec.Resources, r.Spec.Resources)
}
