/*
Copyright 2017 The Stolos Authors.

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

// Package output handles writing Stolos hierarchy objects to the user terminal.
package output

import (
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
)

// PrintForNamespace serializes 'object' to the supplied writer.  It adds a
// tag that denotes the namespace for which the object is printed.
func PrintForNamespace(
	namespace string, object runtime.Object, w io.Writer) error {
	fmt.Fprintf(w, "# Namespace: %q\n", namespace)
	fmt.Fprintf(w, "#\n")
	e := json.NewYAMLSerializer(json.DefaultMetaFactory, nil, nil)
	return e.Encode(object, w)
}
