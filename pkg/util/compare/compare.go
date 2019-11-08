package compare

import (
	"reflect"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/decode"
)

// ObjectMetaEqual returns true if the Meta field of left and right objects are equal.
func ObjectMetaEqual(left core.LabeledAndAnnotated, right core.LabeledAndAnnotated) bool {
	return reflect.DeepEqual(left.GetLabels(), right.GetLabels()) && reflect.DeepEqual(left.GetAnnotations(), right.GetAnnotations())
}

// GenericResourcesEqual returns true if the GenericResources slices are
// equivalent.
// Since the GenericResources in the cluster have the RawExtension.Raw field
// populated and the ones being generated have the RawExtension.Object field
// populated, we need to decode them to have a common representation for
// comparing the underlying resources.
func GenericResourcesEqual(decoder decode.Decoder, l []v1.GenericResources, r []v1.GenericResources,
	cmpOptions ...cmp.Option) (bool, error) {
	lr, lErr := decoder.DecodeResources(l)
	if lErr != nil {
		return false, status.InternalWrap(lErr)
	}

	rr, rErr := decoder.DecodeResources(r)
	if rErr != nil {
		return false, status.InternalWrap(rErr)
	}

	return cmp.Equal(lr, rr, cmpOptions...), nil
}
