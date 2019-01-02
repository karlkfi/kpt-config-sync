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

package vet

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/google/nomos/pkg/api/policyhierarchy"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Codes for each Nomos error.
const (
	ReservedDirectoryNameErrorCode              = "1001"
	DuplicateDirectoryNameErrorCode             = "1002"
	IllegalNamespaceSubdirectoryErrorCode       = "1003"
	IllegalNamespaceSelectorAnnotationErrorCode = "1004"
	UnsyncableClusterObjectErrorCode            = "1005"
	UnsyncableNamespaceObjectErrorCode          = "1006"
	IllegalAbstractNamespaceObjectKindErrorCode = "1007"
	ConflictingResourceQuotaErrorCode           = "1008"
	IllegalNamespaceDeclarationErrorCode        = "1009"
	IllegalAnnotationDefinitionErrorCode        = "1010"
	IllegalLabelDefinitionErrorCode             = "1011"
	NamespaceSelectorMayNotHaveAnnotationCode   = "1012"
	ObjectHasUnknownClusterSelectorCode         = "1013"
	InvalidSelectorErrorCode                    = "1014" // TODO: Must refactor to use properly
	MissingDirectoryErrorCode                   = "1015"
	MissingRepoErrorCode                        = "1017"
	IllegalSubdirectoryErrorCode                = "1018"
	IllegalTopLevelNamespaceErrorCode           = "1019"
	InvalidNamespaceNameErrorCode               = "1020"
	UnknownObjectErrorCode                      = "1021" // Impossible to create consistent example.
	DuplicateSyncGroupKindErrorCode             = "1022"
	IllegalKindInSystemErrorCode                = "1024"
	MultipleRepoDefinitionsErrorCode            = "1025"
	MultipleConfigMapsErrorCode                 = "1026"
	UnsupportedRepoSpecVersionCode              = "1027"
	InvalidDirectoryNameErrorCode               = "1028"
	ObjectNameCollisionErrorCode                = "1029"
	MultipleNamespacesErrorCode                 = "1030"
	MissingObjectNameErrorCode                  = "1031"
	UnknownResourceInSyncErrorCode              = "1032"
	IllegalSystemResourcePlacementErrorCode     = "1033"
	UnsupportedResourceInSyncErrorCode          = "1034"
	IllegalHierarchyModeErrorCode               = "1035"
	InvalidMetadataNameErrorCode                = "1036"
	IllegalKindInClusterregistryErrorCode       = "1037"
	IllegalKindInNamespacesErrorCode            = "1038"
	UnknownResourceVersionInSyncErrorCode       = "1039"
	UndefinedErrorCode                          = "????"

	// Obsolete error codes. Do not reuse.
	//Unused1016 = "1016"
	//Unused1023 = "1023"
)

// Example returns a canonical example to use
func Example(code string) Error {
	switch code {
	case ReservedDirectoryNameErrorCode:
		return ReservedDirectoryNameError{Dir: "reserved"}
	case InvalidNamespaceNameErrorCode:
		return InvalidNamespaceNameError{Source: "namespaces/foo/namespace.yaml", Expected: "foo", Actual: "bar"}
	default:
		panic(errors.Errorf("programmer error: example undefined for %T", code))
	}
}

// Explanation returns documentation about what the bug is, why it occurs, and more information on
// how to fix it than just the error message.
func Explanation(code string) string {
	switch code {
	case ReservedDirectoryNameErrorCode:
		return `
GKE Policy Management defines several
[Reserved Namespaces](../management_flow.md#namespaces), and users may
[specify their own Reserved Namespaces](../system_config.md#reserved-namespaces).
Namespace and Abstract Namespace directories MUST NOT use these reserved names.
To fix:

1.  rename the directory,
1.  remove the directory, or
1.  remove the reserved namespace declaration.
`
	case InvalidNamespaceNameErrorCode:
		return `
A Namespace resource MUST have a metadata.name that matches the name of its
directory. To fix, correct the offending Namespace's metadata.name or its
directory.
`
	default:
		panic(errors.Errorf("programmer error: explanation undefined for %T", code))
	}
}

// Error Defines a Kubernetes Nomos Vet error
// These are GKE Policy Management directory errors which are shown to the user and documented.
type Error interface {
	Error() string
	Code() string
}

// withPrefix formats the start of error messages consistently.
func format(err Error, format string, a ...interface{}) string {
	return fmt.Sprintf("KNV%s: ", err.Code()) + fmt.Sprintf(format, a...)
}

type groupVersionKind schema.GroupVersionKind

// String implements Stringer
func (gvk groupVersionKind) String() string {
	return fmt.Sprintf(
		"group: %[1]s\n"+
			"version: %[2]s\n"+
			"kind: %[3]s",
		gvk.Group, gvk.Version, gvk.Kind)
}

type fileObject struct {
	ast.FileObject
}

// String implements Stringer
func (o fileObject) String() string {
	return fmt.Sprintf("source: %[1]s\n"+
		"metadata.name: %[2]s\n"+
		"%[3]s",
		o.Source, o.Name(), groupVersionKind(o.GetObjectKind().GroupVersionKind()))
}

// ReservedDirectoryNameError represents an illegal usage of a reserved name.
type ReservedDirectoryNameError struct {
	Dir string
}

// Error implements error.
func (e ReservedDirectoryNameError) Error() string {
	return format(e,
		"Directories MUST NOT have reserved namespace names. Rename or remove directory:\n\n"+
			"path: %[1]s\n"+
			"name: %[2]s",
		e.Dir, path.Base(e.Dir))
}

// Code implements Error
func (e ReservedDirectoryNameError) Code() string { return ReservedDirectoryNameErrorCode }

// DuplicateDirectoryNameError represents an illegal duplication of directory names.
type DuplicateDirectoryNameError struct {
	Duplicates []string
}

// Error implements error.
func (e DuplicateDirectoryNameError) Error() string {
	// Ensure deterministic node printing order.
	sort.Strings(e.Duplicates)
	return format(e,
		"Directory names MUST be unique. "+
			"Rename one of these directories:\n\n"+
			"%[1]s",
		strings.Join(e.Duplicates, "\n"))
}

// Code implements Error
func (e DuplicateDirectoryNameError) Code() string { return DuplicateDirectoryNameErrorCode }

// IllegalNamespaceSubdirectoryError represents an illegal child directory of a namespace directory.
type IllegalNamespaceSubdirectoryError struct {
	Child  *ast.TreeNode
	Parent *ast.TreeNode
}

// Error implements error.
func (e IllegalNamespaceSubdirectoryError) Error() string {
	return format(e,
		"A %[1]s directory MUST NOT have subdirectories. "+
			"Restructure %[4]q so that it does not have subdirectory %[2]q:\n\n"+
			"%[3]s",
		ast.Namespace, e.Child.Name(), e.Child, e.Parent.Name())
}

// Code implements Error
func (e IllegalNamespaceSubdirectoryError) Code() string { return IllegalNamespaceSubdirectoryErrorCode }

// IllegalNamespaceSelectorAnnotationError represents an illegal usage of the namespace selector annotation.
type IllegalNamespaceSelectorAnnotationError struct {
	*ast.TreeNode
}

// Error implements error.
func (e IllegalNamespaceSelectorAnnotationError) Error() string {
	return format(e,
		"A %[3]s MUST NOT use the annotation %[2]s. "+
			"Remove metadata.annotations.%[2]s from:\n\n"+
			"%[1]s",
		e.TreeNode, v1alpha1.NamespaceSelectorAnnotationKey, ast.Namespace)
}

// Code implements Error
func (e IllegalNamespaceSelectorAnnotationError) Code() string {
	return IllegalNamespaceSelectorAnnotationErrorCode
}

// UnsyncableClusterObjectError represents an illegal usage of a cluster object kind which has not be explicitly declared.
type UnsyncableClusterObjectError struct {
	*ast.ClusterObject
}

// Error implements error.
func (e UnsyncableClusterObjectError) Error() string {
	return format(e,
		"Unable to sync resource %[2]q. "+
			"Enable sync for this resource's kind.\n\n"+
			"%[1]s",
		fileObject{e.FileObject}, e.Name())
}

// Code implements Error
func (e UnsyncableClusterObjectError) Code() string { return UnsyncableClusterObjectErrorCode }

// UnsyncableNamespaceObjectError represents an illegal usage of a resource which has not been defined for use in namespaces/.
type UnsyncableNamespaceObjectError struct {
	*ast.NamespaceObject
}

// Error implements error.
func (e UnsyncableNamespaceObjectError) Error() string {
	return format(e,
		"Unable to sync resource %[2]q. "+
			"Enable sync for this resource's kind.\n\n"+
			"%[1]s",
		fileObject{e.FileObject}, e.Name())
}

// Code implements Error
func (e UnsyncableNamespaceObjectError) Code() string { return UnsyncableNamespaceObjectErrorCode }

// IllegalAbstractNamespaceObjectKindError represents an illegal usage of a kind not allowed in abstract namespaces.
type IllegalAbstractNamespaceObjectKindError struct {
	*ast.NamespaceObject
}

// Error implements error.
func (e IllegalAbstractNamespaceObjectKindError) Error() string {
	return format(e,
		"Resource %[4]q illegally declared in an %[1]s directory. "+
			"Move this resource to a %[2]s directory:\n\n"+
			"%[3]s",
		ast.AbstractNamespace, ast.Namespace, fileObject{e.FileObject}, e.Name())
}

// Code implements Error
func (e IllegalAbstractNamespaceObjectKindError) Code() string {
	return IllegalAbstractNamespaceObjectKindErrorCode
}

// ConflictingResourceQuotaError represents multiple ResourceQuotas illegally presiding in the same directory.
type ConflictingResourceQuotaError struct {
	Path       string
	Duplicates []ast.FileObject
}

// Error implements error.
func (e ConflictingResourceQuotaError) Error() string {
	var strs []string
	for _, duplicate := range e.Duplicates {
		strs = append(strs, fmt.Sprintf("source: %[1]s\nname: %[2]s",
			path.Join(e.Path, path.Base(duplicate.Source)), duplicate.Name()))
	}
	sort.Strings(strs)

	return format(e,
		"A directory MUST NOT contain more than one ResourceQuota resource. "+
			"Directory %[1]q contains multiple ResourceQuota resources:\n\n"+
			"%[2]s",
		e.Path, strings.Join(strs, "\n\n"))
}

// Code implements Error
func (e ConflictingResourceQuotaError) Code() string { return ConflictingResourceQuotaErrorCode }

// IllegalMetadataNamespaceDeclarationError represents illegally declaring metadata.namespace
type IllegalMetadataNamespaceDeclarationError struct {
	Object ast.FileObject
}

// Error implements error.
func (e IllegalMetadataNamespaceDeclarationError) Error() string {
	// TODO(willbeason): Error unused until b/118715158
	return format(e,
		"Resources MUST NOT declare metadata.namespace:\n\n"+
			"%[1]s",
		fileObject{e.Object})
}

// Code implements Error
func (e IllegalMetadataNamespaceDeclarationError) Code() string {
	return IllegalNamespaceDeclarationErrorCode
}

// IllegalAnnotationDefinitionError represents a set of illegal annotation definitions.
type IllegalAnnotationDefinitionError struct {
	Object      ast.FileObject
	Annotations []string
}

// Error implements error.
func (e IllegalAnnotationDefinitionError) Error() string {
	sort.Strings(e.Annotations) // ensure deterministic annotation order
	a := strings.Join(e.Annotations, ", ")
	return format(e,
		"Resources MUST NOT declare unsupported annotations starting with %[3]q. "+
			"Resource %[4]q has offending annotations: %[1]s\n\n"+
			"%[2]s",
		a, fileObject{e.Object}, policyhierarchy.GroupName, e.Object.Name())
}

// Code implements Error
func (e IllegalAnnotationDefinitionError) Code() string { return IllegalAnnotationDefinitionErrorCode }

// IllegalLabelDefinitionError represent a set of illegal label definitions.
type IllegalLabelDefinitionError struct {
	Object ast.FileObject
	Labels []string
}

// Error implements error.
func (e IllegalLabelDefinitionError) Error() string {
	sort.Strings(e.Labels) // ensure deterministic label order
	l := strings.Join(e.Labels, ", ")
	return format(e,
		"Resources MUST NOT declare labels starting with %[3]q. "+
			"Below resource declares these offending labels: %[1]s\n\n"+
			"%[2]s",
		l, fileObject{e.Object}, policyhierarchy.GroupName)
}

// Code implements Error
func (e IllegalLabelDefinitionError) Code() string { return IllegalLabelDefinitionErrorCode }

// NamespaceSelectorMayNotHaveAnnotation reports that a namespace selector has
// an annotation that is not allowed.
type NamespaceSelectorMayNotHaveAnnotation struct {
	Object metav1.Object
}

// Error implements error.
func (e NamespaceSelectorMayNotHaveAnnotation) Error() string {
	// TODO(willbeason): Print information about the object so it can actually be found.
	return format(e, "The NamespaceSelector resource %q MUST NOT have ClusterSelector annotation", e.Object.GetName())
}

// Code implements Error
func (e NamespaceSelectorMayNotHaveAnnotation) Code() string {
	return NamespaceSelectorMayNotHaveAnnotationCode
}

// ObjectHasUnknownClusterSelector is an error denoting an object that has an unknown annotation.
type ObjectHasUnknownClusterSelector struct {
	Object     metav1.Object
	Annotation string
}

// Error implements error.
func (e ObjectHasUnknownClusterSelector) Error() string {
	return format(e, "Resource %q MUST refer to an existing ClusterSelector, but has annotation %s=%q which maps to no declared ClusterSelector", e.Object.GetName(), v1alpha1.ClusterSelectorAnnotationKey, e.Annotation)
}

// Code implements Error
func (e ObjectHasUnknownClusterSelector) Code() string { return ObjectHasUnknownClusterSelectorCode }

// InvalidSelectorError is a validation error.
type InvalidSelectorError struct {
	Name  string
	Cause error
}

// Error implements error.
func (e InvalidSelectorError) Error() string {
	return format(e, errors.Wrapf(e.Cause, "ClusterSelector %q has validation errors that must be corrected", e.Name).Error())
}

// Code implements Error
func (e InvalidSelectorError) Code() string { return InvalidSelectorErrorCode }

// MissingDirectoryError reports that a required directory is missing.
type MissingDirectoryError struct{}

// Error implements error.
func (e MissingDirectoryError) Error() string {
	return format(e,
		"Required %s/ directory is missing.", repo.SystemDir)
}

// Code implements Error
func (e MissingDirectoryError) Code() string { return MissingDirectoryErrorCode }

// MissingRepoError reports that there is no Repo definition in system/
type MissingRepoError struct{}

// Error implements error
func (e MissingRepoError) Error() string {
	return format(e,
		"%s/ directory must declare a Repo resource.", repo.SystemDir)
}

// Code implements Error
func (e MissingRepoError) Code() string { return MissingRepoErrorCode }

// IllegalSubdirectoryError reports that the directory has an illegal subdirectory.
type IllegalSubdirectoryError struct {
	BaseDir string
	SubDir  string
}

// Error implements error
func (e IllegalSubdirectoryError) Error() string {
	return format(e,
		"%s/ directory MUST NOT have subdirectories.\n\n"+
			"path: %[2]s", e.BaseDir, e.SubDir)
}

// Code implements Error
func (e IllegalSubdirectoryError) Code() string { return IllegalSubdirectoryErrorCode }

// IllegalTopLevelNamespaceError reports that there may not be a Namespace declared directly in namespaces/
type IllegalTopLevelNamespaceError struct {
	Object ast.FileObject
}

// Error implements error
func (e IllegalTopLevelNamespaceError) Error() string {
	return format(e,
		"%[2]ss MUST be declared in subdirectories of %[1]s/. Create a subdirectory for %[2]ss declared in:\n\n"+
			"source: %[3]s",
		repo.NamespacesDir, ast.Namespace, e.Object.Source)
}

// Code implements Error
func (e IllegalTopLevelNamespaceError) Code() string { return IllegalTopLevelNamespaceErrorCode }

// InvalidNamespaceNameError reports that a Namespace has an invalid name.
type InvalidNamespaceNameError struct {
	Source   string
	Expected string
	Actual   string
}

// Error implements error
func (e InvalidNamespaceNameError) Error() string {
	return format(e,
		"%[1]s MUST declare metadata.name that matches the name of its directory.\n\n"+
			"source: %[2]s\n"+
			"expected name: %[3]s\n"+
			"actual name: %[4]s",
		ast.Namespace, e.Source, e.Expected, e.Actual)
}

// Code implements Error
func (e InvalidNamespaceNameError) Code() string { return InvalidNamespaceNameErrorCode }

// UnknownObjectError reports that an object declared in the repo does not have a definition in the cluster.
type UnknownObjectError struct {
	*ast.FileObject
}

// Error implements error
func (e UnknownObjectError) Error() string {
	return format(e,
		"Transient Error: Resource is declared, but has no definition on the cluster."+
			"\nResource must be a native K8S resources or have an associated CustomResourceDefinition:\n\n%s",
		e.FileObject)
}

// Code implements Error
func (e UnknownObjectError) Code() string { return UnknownObjectErrorCode }

// DuplicateSyncGroupKindError reports that multiple versions were declared for the same synced kind
type DuplicateSyncGroupKindError struct {
	// Duplicated is the Group/Kind pair with duplicate definitions in Syncs.
	Duplicated schema.GroupKind
	// Sources is the list of all source files containing Syncs defining the duplicate GroupKind pair.
	Sources []string
}

// Error implements error
func (e DuplicateSyncGroupKindError) Error() string {
	return format(e,
		"A Kind for a given Group may be declared at most once:\n\n"+
			"group: %[1]s\n"+
			"kind: %[2]s\n"+
			"sources: [%[3]s]",
		e.Duplicated.Group, e.Duplicated.Kind, strings.Join(e.Sources, ", "))
}

// Code implements Error
func (e DuplicateSyncGroupKindError) Code() string {
	return DuplicateSyncGroupKindErrorCode
}

// IllegalKindInSystemError reports that an object has been illegally defined in system/
type IllegalKindInSystemError struct {
	Source           string
	GroupVersionKind schema.GroupVersionKind
}

// Error implements error
func (e IllegalKindInSystemError) Error() string {
	return format(e,
		"Resources of this Kind may not be declared in %[2]s/:\n\n"+
			"source: %[3]s\n"+
			"%[1]s",
		groupVersionKind(e.GroupVersionKind), repo.SystemDir, e.Source)
}

// Code implements Error
func (e IllegalKindInSystemError) Code() string {
	return IllegalKindInSystemErrorCode
}

// MultipleRepoDefinitionsError reports that the system/ directory contains multiple Repo declarations.
type MultipleRepoDefinitionsError struct {
	Repos map[*v1alpha1.Repo]string
}

// Error implements error
func (e MultipleRepoDefinitionsError) Error() string {
	var repos []string
	// Sort repos so that output is deterministic.
	for r, source := range e.Repos {
		repos = append(repos, fmt.Sprintf("source: %[1]s\n"+
			"name: %[2]s", source, r.Name))
	}
	sort.Strings(repos)

	return format(e,
		"There MUST NOT be more than one Repo declaration in %[1]s/\n\n"+
			"%[2]s",
		repo.SystemDir, strings.Join(repos, "\n\n"))
}

// Code implements Error
func (e MultipleRepoDefinitionsError) Code() string { return MultipleRepoDefinitionsErrorCode }

// MultipleConfigMapsError reports that system/ declares multiple ConfigMaps.
type MultipleConfigMapsError struct {
	ConfigMaps map[*corev1.ConfigMap]string
}

// Error implements error
func (e MultipleConfigMapsError) Error() string {
	var configMaps []string
	// Sort repos so that output is deterministic.
	for c, source := range e.ConfigMaps {
		configMaps = append(configMaps, fmt.Sprintf("source: %[1]s\n"+
			"name: %[2]s", source, c.Name))
	}
	sort.Strings(configMaps)

	return format(e,
		"There MUST NOT be more than one ConfigMap declaration in %[1]s/\n\n"+
			"%[2]s",
		repo.SystemDir, strings.Join(configMaps, "\n\n"))
}

// Code implements Error
func (e MultipleConfigMapsError) Code() string { return MultipleConfigMapsErrorCode }

// UnsupportedRepoSpecVersion reports that the repo version is not supported.
type UnsupportedRepoSpecVersion struct {
	Source  string
	Name    string
	Version string
}

// Error implements error
func (e UnsupportedRepoSpecVersion) Error() string {
	return format(e,
		"Unsupported Repo spec.version: %[3]q. Must use version \"0.1.0\"\n\n"+
			"source: %[1]s\n"+
			"name: %[2]s",
		e.Source, e.Name, e.Version)
}

// Code implements Error
func (e UnsupportedRepoSpecVersion) Code() string { return UnsupportedRepoSpecVersionCode }

// InvalidDirectoryNameError represents an illegal usage of a reserved name.
type InvalidDirectoryNameError struct {
	Dir string
}

// Error implements error.
func (e InvalidDirectoryNameError) Error() string {
	return format(e,
		"Directories MUST be a valid RFC1123 DNS label. Rename or remove directory:\n\n"+
			"path: %[1]s\n"+
			"name: %[2]s",
		e.Dir, path.Base(e.Dir))
}

// Code implements Error
func (e InvalidDirectoryNameError) Code() string { return InvalidDirectoryNameErrorCode }

// ObjectNameCollisionError reports that multiple objects in the same namespace of the same Kind share a name.
type ObjectNameCollisionError struct {
	Name       string
	Duplicates []ast.FileObject
}

// Error implements error
func (e ObjectNameCollisionError) Error() string {
	var strs []string
	for _, duplicate := range e.Duplicates {
		strs = append(strs, fmt.Sprintf(
			"source: %[1]s\n"+
				"%[2]s\n"+
				"name: %[3]s",
			duplicate.Source, groupVersionKind(duplicate.GroupVersionKind()), duplicate.Name()))
	}
	sort.Strings(strs)

	return format(e,
		"Resources of the same Kind MUST have unique names in the same %[1]s and their parent %[3]ss:\n\n"+
			"%[2]s",
		ast.Namespace, strings.Join(strs, "\n\n"), ast.AbstractNamespace)
}

// Code implements Error
func (e ObjectNameCollisionError) Code() string { return ObjectNameCollisionErrorCode }

// MultipleNamespacesError reports that multiple Namespaces are defined in the same directory.
type MultipleNamespacesError struct {
	Duplicates []ast.FileObject
}

// Error implements error
func (e MultipleNamespacesError) Error() string {
	var strs []string
	for _, duplicate := range e.Duplicates {
		strs = append(strs, fileObject{duplicate}.String())
	}
	sort.Strings(strs)

	return format(e,
		"A directory may declare at most one %[1]s resource:\n\n"+
			"%[2]s",
		ast.Namespace, strings.Join(strs, "\n\n"))
}

// Code implements Error
func (e MultipleNamespacesError) Code() string { return MultipleNamespacesErrorCode }

// MissingObjectNameError reports that an object has no name.
type MissingObjectNameError struct {
	Object ast.FileObject
}

// Error implements error
func (e MissingObjectNameError) Error() string {
	return format(e,
		"Resources must declare metadata.name:\n\n"+
			"%[1]s",
		fileObject{e.Object})
}

// Code implements Error
func (e MissingObjectNameError) Code() string { return MissingObjectNameErrorCode }

// UnknownResourceInSyncError reports that a resource defined in a Sync does not have a definition in the cluster.
type UnknownResourceInSyncError struct {
	Source string
	GVK    schema.GroupVersionKind
}

// Error implements error
func (e UnknownResourceInSyncError) Error() string {
	return format(e,
		"Sync defines a Resource Kind that does not exist on cluster. "+
			"Ensure the Group, Version, and Kind are spelled correctly.\n\n"+
			"source: %[1]s\n"+
			"%[2]s",
		e.Source, groupVersionKind(e.GVK))
}

// Code implements Error
func (e UnknownResourceInSyncError) Code() string { return UnknownResourceInSyncErrorCode }

// IllegalSystemResourcePlacementError reports that a nomos.dev object has been defined outside of system/
type IllegalSystemResourcePlacementError struct {
	Object ast.FileObject
}

// Error implements error
func (e IllegalSystemResourcePlacementError) Error() string {
	return format(e,
		"Resources of the below kind MUST NOT be declared outside %[1]s/:\n"+
			"%[2]s",
		repo.SystemDir, fileObject{e.Object}.String())
}

// Code implements Error
func (e IllegalSystemResourcePlacementError) Code() string {
	return IllegalSystemResourcePlacementErrorCode
}

// UnsupportedResourceInSyncError reports that policy management is unsupported for a resource defined in a Sync.
type UnsupportedResourceInSyncError struct {
	Source string
	GVK    schema.GroupVersionKind
}

// Error implements error
func (e UnsupportedResourceInSyncError) Error() string {
	return format(e,
		"This Resource Kind MUST NOT be declared in a Sync:\n\n"+
			"source: %[1]s\n"+
			"%[2]s",
		e.Source, groupVersionKind(e.GVK).String())
}

// Code implements Error
func (e UnsupportedResourceInSyncError) Code() string { return UnsupportedResourceInSyncErrorCode }

// IllegalHierarchyModeError reports that a Sync is defined with a disallowed hierarchyMode.
type IllegalHierarchyModeError struct {
	Source           string
	GroupVersionKind schema.GroupVersionKind
	HierarchyMode    v1alpha1.HierarchyModeType
	Allowed          map[v1alpha1.HierarchyModeType]bool
}

// Error implements error
func (e IllegalHierarchyModeError) Error() string {
	var allowedStr []string
	for a := range e.Allowed {
		allowedStr = append(allowedStr, string(a))
	}
	return format(e,
		"HierarchyMode %[1]q is not a valid value for this Resource. Allowed values are [%[2]s].\n\n"+
			"source: %[3]s\n"+
			"%[4]s",
		e.HierarchyMode, strings.Join(allowedStr, ","), e.Source, groupVersionKind(e.GroupVersionKind))
}

// Code implements Error
func (e IllegalHierarchyModeError) Code() string { return IllegalHierarchyModeErrorCode }

// InvalidMetadataNameError represents the usage of a non-RFC1123 compliant metadata.name
type InvalidMetadataNameError struct {
	Object ast.FileObject
}

// Error implements error.
func (e InvalidMetadataNameError) Error() string {
	return format(e,
		"Resources MUST define a metadata.name which is a valid RFC1123 DNS subdomain. Rename or remove the resource:\n\n"+
			"%[1]s",
		fileObject{e.Object})
}

// Code implements Error
func (e InvalidMetadataNameError) Code() string { return InvalidMetadataNameErrorCode }

// IllegalKindInClusterregistryError reports that an object has been illegally defined in clusterregistry/
type IllegalKindInClusterregistryError struct {
	Source           string
	GroupVersionKind schema.GroupVersionKind
}

// Error implements error
func (e IllegalKindInClusterregistryError) Error() string {
	return format(e,
		"Resources of the below Kind may not be declared in %[2]s/:\n\n"+
			"source: %[3]s\n"+
			"%[1]s",
		groupVersionKind(e.GroupVersionKind), repo.ClusterRegistryDir, e.Source)
}

// Code implements Error
func (e IllegalKindInClusterregistryError) Code() string {
	return IllegalKindInClusterregistryErrorCode
}

// IllegalKindInNamespacesError reports that an object has been illegally defined in namespaces/
type IllegalKindInNamespacesError struct {
	Source           string
	GroupVersionKind schema.GroupVersionKind
}

// Error implements error
func (e IllegalKindInNamespacesError) Error() string {
	return format(e,
		"Resources of the below Kind may not be declared in %[2]s/:\n\n"+
			"source: %[3]s\n"+
			"%[1]s",
		groupVersionKind(e.GroupVersionKind), repo.NamespacesDir, e.Source)
}

// Code implements Error
func (e IllegalKindInNamespacesError) Code() string {
	return IllegalKindInNamespacesErrorCode
}

// UnknownResourceVersionInSyncError reports that a Sync contains a Group/Kind with an incorrect
// Version.
type UnknownResourceVersionInSyncError struct {
	Source  string
	GVK     schema.GroupVersionKind
	Allowed []string
}

// Error implements error
func (e UnknownResourceVersionInSyncError) Error() string {
	return format(e,
		"Sync defines a Resource Kind with an incorrect Version. "+
			"Known Versions: [%[1]s]\n\n"+
			"source: %[2]s\n"+
			"%[3]s",
		strings.Join(e.Allowed, ", "), e.Source, groupVersionKind(e.GVK))
}

// Code implements Error
func (e UnknownResourceVersionInSyncError) Code() string { return UnknownResourceVersionInSyncErrorCode }
