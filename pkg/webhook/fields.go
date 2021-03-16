package webhook

import (
	"fmt"
	"strings"

	"github.com/google/nomos/pkg/api/configmanagement"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/api/configsync/v1beta1"
	"github.com/google/nomos/pkg/declared"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"
)

// ObjectDiffer can compare two versions of an Object and report on any declared
// fields which were modified between the two.
type ObjectDiffer struct {
	converter *declared.ValueConverter
}

// FieldSet returns a Set of the fields in the given Object.
func (d *ObjectDiffer) FieldSet(obj client.Object) (*fieldpath.Set, error) {
	value, err := d.converter.TypedValue(obj)
	if err != nil {
		return nil, err
	}
	return value.ToFieldSet()
}

// FieldDiff returns a Set of the Object fields which are being modified
// in the given Request that are also marked as fields declared in Git.
func (d *ObjectDiffer) FieldDiff(oldObj, newObj client.Object) (*fieldpath.Set, error) {
	oldValue, err := d.converter.TypedValue(oldObj)
	if err != nil {
		return nil, err
	}
	newValue, err := d.converter.TypedValue(newObj)
	if err != nil {
		return nil, err
	}
	cmp, err := oldValue.Compare(newValue)
	if err != nil {
		return nil, err
	}

	// We only check fields that were modified or removed. We don't care about
	// fields that are added since they will never overlap with declared fields.
	return cmp.Modified.Union(cmp.Removed).Union(cmp.Added), nil
}

var (
	metadata     = "metadata"
	annotations  = ".annotations."
	labels       = ".labels."
	metadataPath = fieldpath.PathElement{FieldName: &metadata}
)

// ConfigSyncMetadata returns all of the metadata fields in the given fieldpath
// Set which are ConfigSync labels or annotations.
func ConfigSyncMetadata(set *fieldpath.Set) *fieldpath.Set {
	metadataSet := set.WithPrefix(metadataPath)

	csSet := fieldpath.NewSet()
	metadataSet.Iterate(func(path fieldpath.Path) {
		s := path.String()
		if strings.HasPrefix(s, annotations) {
			s = s[len(annotations):]
			if strings.HasPrefix(s, configsync.GroupName) ||
				strings.HasPrefix(s, configmanagement.GroupName) ||
				strings.HasPrefix(s, v1beta1.LifecyclePrefix) {
				csSet.Insert(path)
			}
		} else if strings.HasPrefix(s, labels) {
			s = s[len(labels):]
			if s == v1.ManagedByKey {
				csSet.Insert(path)
			}
		}
	})
	return csSet
}

// DeclaredFields returns the declared fields for the given Object.
func DeclaredFields(obj client.Object) (*fieldpath.Set, error) {
	decls, ok := obj.GetAnnotations()[v1alpha1.DeclaredFieldsKey]
	if !ok {
		return nil, fmt.Errorf("%s annotation is missing from %s", v1alpha1.DeclaredFieldsKey, object.RuntimeToObjMeta(obj))
	}

	set := &fieldpath.Set{}
	if err := set.FromJSON(strings.NewReader(decls)); err != nil {
		return nil, err
	}
	return set, nil
}
