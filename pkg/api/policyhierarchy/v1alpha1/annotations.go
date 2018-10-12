/*
Copyright 2018 The Nomos Authors.

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

package v1alpha1

// SourcePathAnnotationKey is an annotation indicates the path in the source of truth where the
// policy was originally declared. Paths are slash-separated and OS-agnostic.
const SourcePathAnnotationKey = "nomos.dev/source-path"

// NamespaceSelectorAnnotationKey is the annotation key set on policy resources that refers to
// name of NamespaceSelector resource.
const NamespaceSelectorAnnotationKey = "nomos.dev/namespace-selector"

// InputAnnotations is a map of annotations that are valid to exist on objects when imported from
// the filesystemn.
var InputAnnotations = map[string]struct{}{
	NamespaceSelectorAnnotationKey: {},
}
