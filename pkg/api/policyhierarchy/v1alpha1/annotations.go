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

import (
	"strings"

	"github.com/google/nomos/pkg/api/policyhierarchy"
)

const (
	// NomosPrefix is the prefix for all Nomos annotations.
	NomosPrefix = policyhierarchy.GroupName + "/"

	// ClusterNameAnnotationKey is the annotation key set on Nomos-managed resources that refers to
	// the name of the cluster that the selectors are applied for.
	ClusterNameAnnotationKey = NomosPrefix + "cluster-name"

	// ClusterSelectorAnnotationKey is the annotation key set on Nomos-managed resources that refers
	// to the name of the ClusterSelector resource.
	ClusterSelectorAnnotationKey = NomosPrefix + "cluster-selector"

	// NamespaceSelectorAnnotationKey is the annotation key set on Nomos-managed resources that refers
	// to name of NamespaceSelector resource.
	NamespaceSelectorAnnotationKey = NomosPrefix + "namespace-selector"

	// SourcePathAnnotationKey is the annotation key representing the relative path from POLICY_DIR
	// where the object was originally declared. Paths are slash-separated and OS-agnostic.
	SourcePathAnnotationKey = NomosPrefix + "source-path"

	// SyncTokenAnnotationKey is the annotation key representing the last version token that a Nomos-
	// managed resource was successfully synced from.
	SyncTokenAnnotationKey = NomosPrefix + "sync-token"
)

// HasNomosPrefix returns true if the string begins with the Nomos annotation prefix.
func HasNomosPrefix(s string) bool {
	return strings.HasPrefix(s, NomosPrefix)
}

// inputAnnotations is a map of annotations that are valid to exist on objects when imported from
// the filesystem.
var inputAnnotations = map[string]bool{
	NamespaceSelectorAnnotationKey: true,
	ClusterSelectorAnnotationKey:   true,
}

// IsInputAnnotation returns true if the annotation is a Nomos input annotation.
func IsInputAnnotation(s string) bool {
	return inputAnnotations[s]
}

var nomosAnnotations = map[string]bool{
	ClusterNameAnnotationKey:       true,
	ClusterSelectorAnnotationKey:   true,
	NamespaceSelectorAnnotationKey: true,
	SourcePathAnnotationKey:        true,
	SyncTokenAnnotationKey:         true,
}

// IsAnnotation returns true if the annotation is a nomos system annotation.
func IsAnnotation(a string) bool {
	return nomosAnnotations[a]
}

// GetClusterSelectorAnnotation returns the value of the cluster selector annotation
// among the given annotations.  If the annotation is not there, "" is returned.
func GetClusterSelectorAnnotation(a map[string]string) string {
	// Looking up in a nil map will also return "".
	return a[ClusterSelectorAnnotationKey]
}
