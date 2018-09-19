/*
Copyright 2017 The Nomos Authors.
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

package filesystem

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
)

// Map of annotation keys to a map of old values to new values.
type annotationTransformer map[string]map[string]string

// nolint: deadcode
func newAnnotationTransformer() annotationTransformer {
	return make(map[string]map[string]string)
}

func (t annotationTransformer) addMappingForKey(key string, mapping map[string]string) {
	t[key] = mapping
}

func (t annotationTransformer) transform(obj interface{}) (interface{}, error) {
	o, err := meta.Accessor(obj)
	if err != nil {
		panic(err)
	}
	a := o.GetAnnotations()
	for k, vOldToNew := range t {
		vOld, ok := a[k]
		if !ok {
			continue
		}
		vNew, ok := vOldToNew[vOld]
		if !ok {
			return nil, fmt.Errorf("invalid annotation value %s=%s", k, vOld)
		}
		a[k] = vNew
	}
	return obj, nil
}
