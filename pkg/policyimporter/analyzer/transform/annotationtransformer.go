/*
Copyright 2017 The CSP Config Management Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package transform

import (
	"fmt"

	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
)

type valueMap map[string]string

// annotate replaces the annotation with 'key' on the annotated object o with
// the given value.  It avoids re-setting the map if resetting the map is not
// needed, so won't instantiate unneeded maps.
func annotate(o ast.Annotated, key, value string) ast.Annotated {
	a := o.GetAnnotations()
	wasNil := a == nil
	if a == nil {
		a = make(map[string]string)
	}
	a[key] = value
	if wasNil {
		o.SetAnnotations(a)
	}
	return o
}

// annotatePopulated is like "annotate", except skips annotate if value is empty.
func annotatePopulated(o ast.Annotated, key, value string) {
	if value == "" {
		return
	}
	annotate(o, key, value)
}

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

func (t annotationTransformer) transform(o ast.Annotated) error {
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
