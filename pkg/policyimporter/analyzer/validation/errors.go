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
	"reflect"
	"sort"
	"strings"

	"github.com/google/nomos/pkg/api/policyhierarchy"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// code returns the unique code associated with the error type.
// Only ever (1) add to this method or (2) deprecate ids. Do not reuse.
func code(e error) string {
	switch e.(type) {
	case ReservedDirectoryNameError:
		return "1001"
	case DuplicateDirectoryNameError:
		return "1002"
	case IllegalNamespaceSubdirectoryError:
		return "1003"
	case IllegalNamespaceSelectorAnnotationError:
		return "1004"
	case UnsyncableClusterObjectError:
		return "1005"
	case UnsyncableNamespaceObjectError:
		return "1006"
	case IllegalAbstractNamespaceObjectKindError:
		return "1007"
	case ConflictingResourceQuotaError:
		return "1008"
	case IllegalNamespaceDeclarationError:
		return "1009"
	case IllegalAnnotationDefinitionError:
		return "1010"
	case IllegalLabelDefinitionError:
		return "1011"
	case NamespaceSelectorMayNotHaveAnnotation:
		return "1012"
	case ObjectHasUnknownClusterSelector:
		return "1013"
	case InvalidSelector:
		return "1014"
	case MissingSystemDirectoryError:
		return "1015"
	case EmptySystemDirectoryError:
		return "1016"
	case MissingRepoError:
		return "1017"
	case IllegalClusterSubdirectoryError:
		return "1018"
	default:
		panic(fmt.Sprintf("Unknown Nomosvet Error Type: %T", reflect.TypeOf(e))) // Undefined
	}
}

// withPrefix formats the start of error messages consistently.
func format(err error, format string, a ...interface{}) string {
	return fmt.Sprintf("KNV%s: ", code(err)) + fmt.Sprintf(format, a...)
}

// ReservedDirectoryNameError represents an illegal usage of a reserved name.
type ReservedDirectoryNameError struct {
	*ast.TreeNode
}

// Error implements error.
func (e ReservedDirectoryNameError) Error() string {
	return format(e,
		"Directories MUST NOT have reserved namespace names. "+
			"Rename or remove directory %[1]q from %[4]s/%[3]s:\n\n"+
			"%[2]s",
		e.Name(), e.TreeNode, v1alpha1.ReservedNamespacesConfigMapName, repo.SystemDir)
}

// DuplicateDirectoryNameError represents an illegal duplication of directory names.
type DuplicateDirectoryNameError struct {
	this  *ast.TreeNode
	other *ast.TreeNode
}

// Error implements error.
func (e DuplicateDirectoryNameError) Error() string {
	// Ensure deterministic node printing order for n = 2
	// For n >= 3, we can't be sure the canonical "first" directory will be one of the two presented
	// to the user. So we can't guarantee determinism for n >= 3.
	var first, second = e.this, e.other
	if first.Path > second.Path {
		first, second = e.other, e.this
	}
	return format(e,
		"Directory names MUST be unique. "+
			"Rename one of these two directories:\n\n"+
			"%[1]s\n\n"+
			"%[2]s",
		first, second)
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
		e.FileObject, e.Name())
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
		e.FileObject, e.Name())
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
		ast.AbstractNamespace, ast.Namespace, e.FileObject, e.Name())
}

// ConflictingResourceQuotaError represents multiple ResourceQuotas illegally presiding in the same directory.
type ConflictingResourceQuotaError struct {
	*ast.NamespaceObject
}

// Error implements error.
func (e ConflictingResourceQuotaError) Error() string {
	return format(e,
		"A directory MUST NOT contain more than one ResourceQuota object. "+
			"Directory %[1]q contains multiple ResourceQuota objects, including:\n\n"+
			"%[2]s",
		path.Dir(e.Source), e.FileObject)
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
			"Object %[4]q has offending annotations: %[1]s\n\n%[2]s",
		a, e.object, policyhierarchy.GroupName, e.object.Name())
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
			"Below object defines these offending labels: %[1]s\n\n%[2]s",
		l, e.object, policyhierarchy.GroupName)
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

// MissingSystemDirectoryError reports that the required system/ directory is missing.
type MissingSystemDirectoryError struct{}

// Error implements error.
func (e MissingSystemDirectoryError) Error() string {
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

// IllegalClusterSubdirectoryError reports that the cluster/ directory has an illegal subdirectory.
type IllegalClusterSubdirectoryError struct {
	subdirectory string
}

// Error implements error
func (e IllegalClusterSubdirectoryError) Error() string {
	return format(e,
		"%s/ directory MUST NOT have subdirectories.\n\npath: %[2]s", repo.ClusterDir, e.subdirectory)
}
