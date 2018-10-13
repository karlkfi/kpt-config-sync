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

const (
	// SourcePathAnnotationKey is the annotation key representing the relative path from POLICY_DIR
	// where the object was originally declared. Paths are slash-separated and OS-agnostic.
	SourcePathAnnotationKey = "nomos.dev/source-path"

	// NamespaceSelectorAnnotationKey is the annotation key set on policy resources that refers to
	// name of NamespaceSelector resource.
	NamespaceSelectorAnnotationKey = "nomos.dev/namespace-selector"

	// ClusterSelectorAnnotationKey is the annotation key set on policy resources that refers to the name of the ClusterSelector resource.
	ClusterSelectorAnnotationKey = "nomos.dev/cluster-selector"

	// ClusterNameAnnotationKey is the annotation key set on policy resources that refers to the name of the cluster that the selectors are applied for.
	ClusterNameAnnotationKey = "nomos.dev/cluster-name"
)

// InputAnnotations is a map of annotations that are valid to exist on objects when imported from
// the filesystem.
var InputAnnotations = map[string]struct{}{
	NamespaceSelectorAnnotationKey: {},
	ClusterSelectorAnnotationKey:   {},
}

var nomosAnnotations = map[string]bool{
	NamespaceSelectorAnnotationKey: true,
	SourcePathAnnotationKey:        true,
	ClusterSelectorAnnotationKey:   true,
	ClusterNameAnnotationKey:       true,
}

// IsAnnotation returns true if the annotation is a nomos system annotation.
func IsAnnotation(a string) bool {
	return nomosAnnotations[a]
}
