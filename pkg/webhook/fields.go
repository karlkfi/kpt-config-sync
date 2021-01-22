package webhook

import (
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
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

// DeclaredFieldDiff returns a Set of the Object fields which are being modified
// in the given Request that are also marked as fields declared in Git.
func (d *ObjectDiffer) DeclaredFieldDiff(oldObj, newObj client.Object) (s *fieldpath.Set, err error) {
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

	declared, err := declaredFields(oldObj)
	if err != nil {
		return nil, err
	}

	// We only check fields that were modified or removed. We don't care about
	// fields that are added since they will never overlap with declared fields.
	return cmp.Modified.Union(cmp.Removed).Intersection(declared), nil
}

// declaredFields returns the declared fields for the given Object.
func declaredFields(obj client.Object) (s *fieldpath.Set, err error) {
	declared, ok := obj.GetAnnotations()[v1alpha1.DeclaredFieldsKey]
	if !ok {
		if err != nil {
			glog.Errorf("Failed to get object metadata: %v", err)
		}
		return nil, fmt.Errorf("%s annotation is missing from %s", v1alpha1.DeclaredFieldsKey, object.RuntimeToObjMeta(obj))
	}

	set := &fieldpath.Set{}
	if err := set.FromJSON(strings.NewReader(declared)); err != nil {
		return nil, err
	}
	return set, nil
}
