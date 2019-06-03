package clusterconfig

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var ccsIgnore = []cmp.Option{
	cmpopts.IgnoreFields(v1.ClusterConfigSpec{}, "Token", "ImportTime"),
}

// ClusterConfigsEqual returns true if the clusterconfigs are equivalent.
func ClusterConfigsEqual(lhs runtime.Object, rhs runtime.Object) bool {
	l := lhs.(*v1.ClusterConfig)
	r := rhs.(*v1.ClusterConfig)
	return cmp.Equal(l.Spec, r.Spec, ccsIgnore...)
}
