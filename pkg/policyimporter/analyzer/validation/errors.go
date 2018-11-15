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

package validation

import (
	"fmt"
	"path"
	"path/filepath"
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
	"k8s.io/kubernetes/pkg/kubectl/genericclioptions/resource"
)

// Codes for each Nomos error.
const (
	ReservedDirectoryNameErrorCode                 = "1001"
	DuplicateDirectoryNameErrorCode                = "1002"
	IllegalNamespaceSubdirectoryErrorCode          = "1003"
	IllegalNamespaceSelectorAnnotationErrorCode    = "1004"
	UnsyncableClusterObjectErrorCode               = "1005"
	UnsyncableNamespaceObjectErrorCode             = "1006"
	IllegalAbstractNamespaceObjectKindErrorCode    = "1007"
	ConflictingResourceQuotaErrorCode              = "1008"
	IllegalNamespaceDeclarationErrorCode           = "1009" // TODO(willbeason): Unused
	IllegalAnnotationDefinitionErrorCode           = "1010"
	IllegalLabelDefinitionErrorCode                = "1011"
	NamespaceSelectorMayNotHaveAnnotationCode      = "1012"
	ObjectHasUnknownClusterSelectorCode            = "1013"
	InvalidSelectorCode                            = "1014" // TODO: Add tests in parser_test.go
	MissingSystemDirectoryErrorCode                = "1015"
	EmptySystemDirectoryErrorCode                  = "1016" // TODO(willbeason): Unused
	MissingRepoErrorCode                           = "1017"
	IllegalSubdirectoryErrorCode                   = "1018"
	IllegalTopLevelNamespaceErrorCode              = "1019"
	InvalidNamespaceNameErrorCode                  = "1020"
	UnknownObjectErrorCode                         = "1021"
	MultipleVersionForSameSyncedTypeErrorCode      = "1022"
	IllegalNamespaceSyncDeclarationErrorCode       = "1023"
	IllegalSystemObjectDefinitionInSystemErrorCode = "1024"
	MultipleRepoDefinitionsErrorCode               = "1025"
	MultipleConfigMapsErrorCode                    = "1026"
	UnsupportedRepoSpecVersionCode                 = "1027"
	InvalidDirectoryNameErrorCode                  = "1028"
	ObjectNameCollisionErrorCode                   = "1029"
	MultipleNamespacesErrorCode                    = "1030"
	MissingObjectNameErrorCode                     = "1031"
	UnknownResourceInSyncErrorCode                 = "1032"
	UndefinedErrorCode                             = "????"
)

// Code returns the unique code associated with the error type.
// Only ever (1) add to this method or (2) deprecate ids. Do not reuse.
func Code(e error) string {
	switch e.(type) {
	case ReservedDirectoryNameError:
		return ReservedDirectoryNameErrorCode
	case DuplicateDirectoryNameError:
		return DuplicateDirectoryNameErrorCode
	case IllegalNamespaceSubdirectoryError:
		return IllegalNamespaceSubdirectoryErrorCode
	case IllegalNamespaceSelectorAnnotationError:
		return IllegalNamespaceSelectorAnnotationErrorCode
	case UnsyncableClusterObjectError:
		return UnsyncableClusterObjectErrorCode
	case UnsyncableNamespaceObjectError:
		return UnsyncableNamespaceObjectErrorCode
	case IllegalAbstractNamespaceObjectKindError:
		return IllegalAbstractNamespaceObjectKindErrorCode
	case ConflictingResourceQuotaError:
		return ConflictingResourceQuotaErrorCode
	case IllegalNamespaceDeclarationError:
		return IllegalNamespaceDeclarationErrorCode
	case IllegalAnnotationDefinitionError:
		return IllegalAnnotationDefinitionErrorCode
	case IllegalLabelDefinitionError:
		return IllegalLabelDefinitionErrorCode
	case NamespaceSelectorMayNotHaveAnnotation:
		return NamespaceSelectorMayNotHaveAnnotationCode
	case ObjectHasUnknownClusterSelector:
		return ObjectHasUnknownClusterSelectorCode
	case InvalidSelector:
		return InvalidSelectorCode
	case MissingDirectoryError:
		return MissingSystemDirectoryErrorCode
	case EmptySystemDirectoryError:
		return EmptySystemDirectoryErrorCode
	case MissingRepoError:
		return MissingRepoErrorCode
	case IllegalSubdirectoryError:
		return IllegalSubdirectoryErrorCode
	case IllegalTopLevelNamespaceError:
		return IllegalTopLevelNamespaceErrorCode
	case InvalidNamespaceNameError:
		return InvalidNamespaceNameErrorCode
	case UnknownObjectError:
		return UnknownObjectErrorCode
	case MultipleVersionForSameSyncedTypeError:
		return MultipleVersionForSameSyncedTypeErrorCode
	case IllegalNamespaceSyncDeclarationError:
		return IllegalNamespaceSyncDeclarationErrorCode
	case IllegalSystemObjectDefinitionInSystemError:
		return IllegalSystemObjectDefinitionInSystemErrorCode
	case MultipleRepoDefinitionsError:
		return MultipleRepoDefinitionsErrorCode
	case MultipleConfigMapsError:
		return MultipleConfigMapsErrorCode
	case UnsupportedRepoSpecVersion:
		return UnsupportedRepoSpecVersionCode
	case InvalidDirectoryNameError:
		return InvalidDirectoryNameErrorCode
	case ObjectNameCollisionError:
		return ObjectNameCollisionErrorCode
	case MultipleNamespacesError:
		return MultipleNamespacesErrorCode
	case MissingObjectNameError:
		return MissingObjectNameErrorCode
	case UnknownResourceInSyncError:
		return UnknownResourceInSyncErrorCode
	default:
		return UndefinedErrorCode // Undefined
	}
}

// withPrefix formats the start of error messages consistently.
func format(err error, format string, a ...interface{}) string {
	code := Code(err)
	if code == UndefinedErrorCode {
		// Only reachable by programmer error. Requires calling format() on an error other than the ones
		// defined in this file or not having an entry in Code() above.
		panic(fmt.Sprintf("Unknown Nomosvet Error: %s", err.Error()))
	}
	return fmt.Sprintf("KNV%s: ", Code(err)) + fmt.Sprintf(format, a...)
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

// IllegalNamespaceSubdirectoryError represents an illegal child directory of a namespace directory.
type IllegalNamespaceSubdirectoryError struct {
	child  *ast.TreeNode
	parent *ast.TreeNode
}

// Error implements error.
func (e IllegalNamespaceSubdirectoryError) Error() string {
	return format(e,
		"A %[1]s directory MUST NOT have subdirectories. "+
			"Restructure %[4]q so that it does not have subdirectory %[2]q:\n\n"+
			"%[3]s",
		ast.Namespace, e.child.Name(), e.child, e.parent.Name())
}

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

// UnsyncableClusterObjectError represents an illegal usage of a cluster object kind which has not be explicitly declared.
type UnsyncableClusterObjectError struct {
	*ast.ClusterObject
}

// Error implements error.
func (e UnsyncableClusterObjectError) Error() string {
	return format(e,
		"Unable to sync cluster object %[2]q. "+
			"Enable sync for this object's kind.\n\n"+
			"%[1]s",
		fileObject{e.FileObject}, e.Name())
}

// UnsyncableNamespaceObjectError represents an illegal usage of a namespace object kind which has not been explicitly declared.
type UnsyncableNamespaceObjectError struct {
	*ast.NamespaceObject
}

// Error implements error.
func (e UnsyncableNamespaceObjectError) Error() string {
	return format(e,
		"Unable to sync namespace object %[2]q. "+
			"Enable sync for this object's kind.\n\n"+
			"%[1]s",
		fileObject{e.FileObject}, e.Name())
}

// IllegalAbstractNamespaceObjectKindError represents an illegal usage of a kind not allowed in abstract namespaces.
type IllegalAbstractNamespaceObjectKindError struct {
	*ast.NamespaceObject
}

// Error implements error.
func (e IllegalAbstractNamespaceObjectKindError) Error() string {
	return format(e,
		"Object %[4]q illegally declared in an %[1]s directory. "+
			"Move this object to a %[2]s directory:\n\n"+
			"%[3]s",
		ast.AbstractNamespace, ast.Namespace, fileObject{e.FileObject}, e.Name())
}

// ConflictingResourceQuotaError represents multiple ResourceQuotas illegally presiding in the same directory.
type ConflictingResourceQuotaError struct {
	Path       string
	Duplicates []*resource.Info
}

// Error implements error.
func (e ConflictingResourceQuotaError) Error() string {
	strs := []string{}
	for _, duplicate := range e.Duplicates {
		strs = append(strs, fmt.Sprintf("source: %[1]s\nname: %[2]s",
			path.Join(e.Path, path.Base(duplicate.Source)), duplicate.Name))
	}
	sort.Strings(strs)

	return format(e,
		"A directory MUST NOT contain more than one ResourceQuota object. "+
			"Directory %[1]q contains multiple ResourceQuota objects:\n\n"+
			"%[2]s",
		e.Path, strings.Join(strs, "\n\n"))
}

// IllegalNamespaceDeclarationError represents illegally declaring metadata.namespace
type IllegalNamespaceDeclarationError struct {
	*ast.NamespaceObject
}

// Error implements error.
func (e IllegalNamespaceDeclarationError) Error() string {
	// TODO(willbeason): Error unused until b/118715158
	return format(e,
		"Objects MUST NOT delcare metadata.namespace. "+
			"Object %[2]q declares metadata.namespace:\n\n"+
			"%[1]s",
		e.FileObject, e.Name())
}

// IllegalAnnotationDefinitionError represents a set of illegal annotation definitions.
type IllegalAnnotationDefinitionError struct {
	object      ast.FileObject
	annotations []string
}

// Error implements error.
func (e IllegalAnnotationDefinitionError) Error() string {
	sort.Strings(e.annotations) // ensure deterministic annotation order
	a := strings.Join(e.annotations, ", ")
	return format(e,
		"Objects MUST NOT define unsupported annotations starting with %[3]q. "+
			"Object %[4]q has offending annotations: %[1]s\n\n"+
			"%[2]s",
		a, fileObject{e.object}, policyhierarchy.GroupName, e.object.Name())
}

// IllegalLabelDefinitionError represent a set of illegal label definitions.
type IllegalLabelDefinitionError struct {
	object ast.FileObject
	labels []string
}

// Error implements error.
func (e IllegalLabelDefinitionError) Error() string {
	sort.Strings(e.labels) // ensure deterministic label order
	l := strings.Join(e.labels, ", ")
	return format(e,
		"Objects MUST NOT define labels starting with %[3]q. "+
			"Below object defines these offending labels: %[1]s\n\n"+
			"%[2]s",
		l, fileObject{e.object}, policyhierarchy.GroupName)
}

// NamespaceSelectorMayNotHaveAnnotation reports that a namespace selector has
// an annotation that is not allowed.
type NamespaceSelectorMayNotHaveAnnotation struct {
	o metav1.Object
}

// Error implements error.
func (e NamespaceSelectorMayNotHaveAnnotation) Error() string {
	return format(e, "The NamespaceSelector object %q in namespace %q MUST NOT have ClusterSelector annotation", e.o.GetName(), e.o.GetNamespace())
}

// ObjectHasUnknownClusterSelector is an error denoting an object that has an unknown annotation.
type ObjectHasUnknownClusterSelector struct {
	o metav1.Object
	a string // The annotation that the object used.
}

// Error implements error.
func (e ObjectHasUnknownClusterSelector) Error() string {
	return format(e, "Object %q in namespace %q MUST refer to an existing ClusterSelector, but has annotation %s=%q which maps to no defined ClusterSelector", e.o.GetName(), e.o.GetNamespace(), v1alpha1.ClusterSelectorAnnotationKey, e.a)
}

// InvalidSelector is a validation error.
type InvalidSelector struct {
	name  string
	cause error
}

// Error implements error.
func (e InvalidSelector) Error() string {
	return format(e, errors.Wrapf(e.cause, "ClusterSelector %q has validation errors that must be corrected", e.name).Error())
}

// MissingDirectoryError reports that a required directory is missing.
type MissingDirectoryError struct{}

// Error implements error.
func (e MissingDirectoryError) Error() string {
	return format(e,
		"Required %s/ directory is missing.", repo.SystemDir)
}

// EmptySystemDirectoryError reports that the system/ directory is empty.
type EmptySystemDirectoryError struct{}

// Error implements error.
func (e EmptySystemDirectoryError) Error() string {
	return format(e,
		"%s/ directory must have at least one file, defining a Repo object.", repo.SystemDir)
}

// MissingRepoError reports that there is no Repo definition in system/
type MissingRepoError struct{}

// Error implements error
func (e MissingRepoError) Error() string {
	return format(e,
		"%s/ directory must define an object of type Repo.", repo.SystemDir)
}

// IllegalSubdirectoryError reports that the directory has an illegal subdirectory.
type IllegalSubdirectoryError struct {
	Dir    string
	SubDir string
}

// Error implements error
func (e IllegalSubdirectoryError) Error() string {
	dir := path.Base(e.Dir)
	relpath, _ := filepath.Rel(e.Dir, e.SubDir)
	return format(e,
		"%s/ directory MUST NOT have subdirectories.\n\n"+
			"path: %[2]s", dir, path.Join(dir, relpath))
}

// IllegalTopLevelNamespaceError reports that there may not be a Namespace declared directly in namespaces/
type IllegalTopLevelNamespaceError struct {
	Source string
	Info   *resource.Info
}

// Error implements error
func (e IllegalTopLevelNamespaceError) Error() string {
	return format(e,
		"%[2]ss MUST be declared in subdirectories of %[1]s/. Create a subdirectory for namespaces in:\n\n"+
			"source: %[3]s",
		repo.NamespacesDir, ast.Namespace, e.Source)
}

// InvalidNamespaceNameError reports that a Namespace has an invalid name.
type InvalidNamespaceNameError struct {
	Source   string
	Expected string
	Actual   string
}

// Error implements error
func (e InvalidNamespaceNameError) Error() string {
	return format(e,
		"%[1]s MUST define metadata.name that matches the name of its directory.\n\n"+
			"source: %[2]s\n"+
			"expected name: %[3]s\n"+
			"actual name: %[4]s",
		ast.Namespace, e.Source, e.Expected, e.Actual)
}

// UnknownObjectError reports that an object declared in the repo does not have a definition in the cluster.
type UnknownObjectError struct {
	*ast.FileObject
}

// Error implements error
func (e UnknownObjectError) Error() string {
	return format(e,
		"Transient Error: Object is declared, but has no definition on the cluster."+
			"\nObject must be a native K8S objects or have an associated CustomResourceDefinition:\n\n%s",
		e.FileObject)
}

// MultipleVersionForSameSyncedTypeError reports that multiple versions were declared for the same synced kind
type MultipleVersionForSameSyncedTypeError struct {
	Source string
	Group  v1alpha1.SyncGroup
	Kind   v1alpha1.SyncKind
}

// PrettyPrint returns a convenient representation of a list of SyncVersions for error messages.
func PrettyPrint(versions []v1alpha1.SyncVersion) string {
	result := make([]string, len(versions))

	return "versions: [" + strings.Join(result, ", ") + "]"
}

// Error implements error
func (e MultipleVersionForSameSyncedTypeError) Error() string {
	return format(e,
		"Kinds MUST declare exactly one version:\n\n"+
			"source: %[1]s\n"+
			"group: %[3]s"+
			"%[4]s\n"+
			"kind: %[2]s",
		e.Source, e.Kind.Kind, e.Group.Group, PrettyPrint(e.Kind.Versions))
}

// IllegalNamespaceSyncDeclarationError reports that Namespace has incorrectly been declared as a Sync type
type IllegalNamespaceSyncDeclarationError struct {
	Source string
}

// Error implements error
func (e IllegalNamespaceSyncDeclarationError) Error() string {
	return format(e,
		"Sync may not declare objects of type %[1]s\n\n"+
			"source: %[2]s",
		ast.Namespace, e.Source)
}

// IllegalSystemObjectDefinitionInSystemError reports that an object has been illegally defined in system/
type IllegalSystemObjectDefinitionInSystemError struct {
	Source           string
	GroupVersionKind schema.GroupVersionKind
}

// Error implements error
func (e IllegalSystemObjectDefinitionInSystemError) Error() string {
	return format(e,
		"Objects of kind %[1]s may not be declared in %[2]s/\n\n"+
			"source: %[3]s\n"+
			"%[1]s",
		groupVersionKind(e.GroupVersionKind), repo.SystemDir, e.Source)
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
		"There MUST NOT be more than one Repo definition in %[1]s/\n\n"+
			"%[2]s",
		repo.SystemDir, strings.Join(repos, "\n\n"))
}

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
		"There MUST NOT be more than one ConfigMap definition in %[1]s/\n\n"+
			"%[2]s",
		repo.SystemDir, strings.Join(configMaps, "\n\n"))
}

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

func relPath(root string, path string) string {
	relPath, err := filepath.Rel(root, path)
	if err != nil {
		panic(errors.Wrap(err, "Tried to process file not in repository."))
	}
	return relPath
}

// ObjectNameCollisionError reports that multiple objects in the same namespace of the same Kind share a name.
type ObjectNameCollisionError struct {
	Name       string
	RootPath   string
	Duplicates []*resource.Info
}

// Error implements error
func (e ObjectNameCollisionError) Error() string {
	strs := []string{}
	for _, duplicate := range e.Duplicates {
		strs = append(strs, fmt.Sprintf(
			"source: %[1]s\n"+
				"%[2]s\n"+
				"name: %[3]s",
			relPath(e.RootPath, duplicate.Source), groupVersionKind(duplicate.Mapping.GroupVersionKind), duplicate.Name))
	}
	sort.Strings(strs)

	return format(e,
		"Objects of the same Kind MUST have unique names in the same %[1]s and their parent %[3]ss:\n\n"+
			"%[2]s",
		ast.Namespace, strings.Join(strs, "\n\n"), ast.AbstractNamespace)
}

// MultipleNamespacesError reports that multiple Namespaces are defined in the same directory.
type MultipleNamespacesError struct {
	Path       string
	RootPath   string
	Duplicates []*resource.Info
}

// Error implements error
func (e MultipleNamespacesError) Error() string {
	strs := []string{}
	for _, duplicate := range e.Duplicates {
		strs = append(strs, fmt.Sprintf(
			"source: %[1]s\n"+
				"%[2]s\n"+
				"name: %[3]s",
			relPath(e.RootPath, duplicate.Source), groupVersionKind(duplicate.Mapping.GroupVersionKind), duplicate.Name))
	}
	sort.Strings(strs)

	return format(e,
		"A directory may declare at most one %[1]s object:\n\n"+
			"%[2]s",
		ast.Namespace, strings.Join(strs, "\n\n"))
}

// MissingObjectNameError reports that an object has no name.
type MissingObjectNameError struct {
	Relpath string
	*resource.Info
}

// Error implements error
func (e MissingObjectNameError) Error() string {
	return format(e,
		"Objects must define metadata.name:\n\n"+
			"source: %[1]s\n"+
			"%[2]s\n"+
			"name: %[3]s",
		e.Relpath, groupVersionKind(e.Mapping.GroupVersionKind), e.Name)
}

// UnknownResourceInSyncError reports that a resource defined on a sync does not have a definition in the cluster.
type UnknownResourceInSyncError struct {
	SyncPath     string
	ResourceType schema.GroupVersionKind
}

// Error implements error
func (e UnknownResourceInSyncError) Error() string {
	return format(e,
		"Sync contains a resource type that does not exist on cluster.\n"+
			"Either remove the resource type from the Sync or create a CustomResourceDefinition for "+
			"the resource type on the cluster.\n\n"+
			"source: %[1]s\n"+
			"%[2]s",
		e.SyncPath, groupVersionKind(e.ResourceType))
}
