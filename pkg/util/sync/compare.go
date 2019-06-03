package sync

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/util/compare"
	"k8s.io/apimachinery/pkg/runtime"
)

// SyncsEqual returns true if the syncs are equivalent.
func SyncsEqual(lhs runtime.Object, rhs runtime.Object) bool {
	l := lhs.(*v1.Sync)
	r := rhs.(*v1.Sync)
	return cmp.Equal(l.Spec, r.Spec) && compare.ObjectMetaEqual(l, r)
}
