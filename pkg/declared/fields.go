package declared

import (
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"
)

// identityFields are the fields in an object which identify it and therefore
// would never mutate.
var identityFields = fieldpath.NewSet(
	fieldpath.MakePathOrDie("apiVersion"),
	fieldpath.MakePathOrDie("kind"),
	fieldpath.MakePathOrDie("metadata"),
	fieldpath.MakePathOrDie("metadata", "name"),
	fieldpath.MakePathOrDie("metadata", "namespace"),
)

// FieldConverter encodes and decodes the fields of an object to/from a format
// that is compatible with the structured-merge-diff of server-side apply.
type FieldConverter struct {
	converter *ValueConverter
}

// EncodeDeclaredFields encodes the fields of the given object into a format that
// is compatible with server-side apply.
func (f *FieldConverter) EncodeDeclaredFields(obj runtime.Object) ([]byte, error) {
	val, err := f.converter.TypedValue(obj)
	if err != nil {
		return nil, err
	}
	set, err := val.ToFieldSet()
	if err != nil {
		return nil, err
	}
	// Strip identity fields away since changing them would change the identity of
	// the object.
	set = set.Difference(identityFields)
	return set.ToJSON()
}
