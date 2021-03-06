package hydrate

import (
	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/objects"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"
)

// DeclaredFields hydrates the given Raw objects by annotating each object with
// its fields that are declared in Git. This annotation is what enables the
// Config Sync admission controller webhook to protect these declared fields
// from being changed by another controller or user.
func DeclaredFields(objs *objects.Raw) status.MultiError {
	if objs.Converter == nil {
		glog.Warning("Skipping declared field hydration. This should only happen for offline executions of nomos vet/hydrate/init.")
		return nil
	}

	var errs status.MultiError
	for _, obj := range objs.Objects {
		fields, err := encodeDeclaredFields(objs.Converter, obj.Object)
		if err != nil {
			errs = status.Append(errs, status.InternalErrorf("failed to encode declared fields: %v", err))
		}
		core.SetAnnotation(obj, v1alpha1.DeclaredFieldsKey, string(fields))
	}
	return errs
}

// identityFields are the fields in an object which identify it and therefore
// would never mutate.
var identityFields = fieldpath.NewSet(
	fieldpath.MakePathOrDie("apiVersion"),
	fieldpath.MakePathOrDie("kind"),
	fieldpath.MakePathOrDie("metadata"),
	fieldpath.MakePathOrDie("metadata", "name"),
	fieldpath.MakePathOrDie("metadata", "namespace"),
	// TODO(b/181994737): Remove the following fields. They should never be
	// allowed in Git, but currently our unit test fakes can generate them so we
	// need to sanitize them until we have more Unstructured fakes for unit tests.
	fieldpath.MakePathOrDie("metadata", "creationTimestamp"),
)

// encodeDeclaredFields encodes the fields of the given object into a format that
// is compatible with server-side apply.
func encodeDeclaredFields(converter *declared.ValueConverter, obj runtime.Object) ([]byte, error) {
	val, err := converter.TypedValue(obj)
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
