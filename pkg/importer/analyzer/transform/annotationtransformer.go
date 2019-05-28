package transform

import (
	"fmt"

	"github.com/google/nomos/pkg/object"
)

type valueMap map[string]string

// annotationTransformer is a map of annotation keys to a map of old values to new values.
//
// Example:
//    t := annotationTransformer{}
//    t.addMappingForKey("myannotation", valueMap{"oldval": "newval"})
//    err := t.transform(object)
//
type annotationTransformer map[string]valueMap

func (t annotationTransformer) addMappingForKey(key string, mapping valueMap) {
	t[key] = mapping
}

func (t annotationTransformer) transform(o object.Annotated) error {
	a := o.GetAnnotations()
	for k, vOldToNew := range t {
		vOld, ok := a[k]
		if !ok {
			continue
		}
		vNew, ok := vOldToNew[vOld]
		if !ok {
			return fmt.Errorf("unrecognized annotation value %s=%s", k, vOld)
		}
		a[k] = vNew
	}
	return nil
}
