package validation

import (
	"fmt"
	"path"
	"strings"

	"github.com/google/nomos/pkg/api/policyhierarchy"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ReservedDirectoryNameError represents an illegal usage of a reserved name.
type ReservedDirectoryNameError struct {
	*ast.TreeNode
}

// Error implements error.
func (e ReservedDirectoryNameError) Error() string {
	return fmt.Sprintf(
		"KNV1001: Directories MUST NOT have reserved namespace names. "+
			"Rename or remove directory %[1]q from %[4]s/%[3]s:\n"+
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
	return fmt.Sprintf(
		"KNV1002: Directory names MUST be unique. "+
			"Rename one of these two directories:\n"+
			"%[1]s\n\n"+
			"%[2]s",
		e.this, e.other)
}

// IllegalNamespaceChildDirectoryError represents an illegal child directory of a namespace directory.
type IllegalNamespaceChildDirectoryError struct {
	child  *ast.TreeNode
	parent *ast.TreeNode
}

// Error implements error.
func (e IllegalNamespaceChildDirectoryError) Error() string {
	return fmt.Sprintf(
		"KNV1003: A %[1]s directory MUST NOT have children. "+
			"Restructure %[4]s so that it does not have child %[2]q:\n"+
			"%[3]s",
		ast.Namespace, e.child.Name(), e.child, e.parent.Name())
}

// IllegalNamespaceSelectorAnnotationError represents an illegal usage of the namespace selector annotation.
type IllegalNamespaceSelectorAnnotationError struct {
	*ast.TreeNode
}

// Error implements error.
func (e IllegalNamespaceSelectorAnnotationError) Error() string {
	return fmt.Sprintf(
		"KNV1004: A %[3]s MUST NOT use the annotation %[2]s. "+
			"Remove metadata.annotations.%[2]s from:\n"+
			"%[1]s",
		e.TreeNode, v1alpha1.NamespaceSelectorAnnotationKey, ast.Namespace)
}

// UnsyncableClusterObjectError represents an illegal usage of a cluster object kind which has not be explicitly declared.
type UnsyncableClusterObjectError struct {
	*ast.ClusterObject
}

// Error implements error.
func (e UnsyncableClusterObjectError) Error() string {
	return fmt.Sprintf(
		"KNV1005: Object %[4]q defined in %[1]s/ is not syncable. "+
			"Enable sync for objects of kind %[2]s.\n"+
			"%[3]s",
		repo.ClusterDir, e.GroupVersionKind(), e.FileObject, e.Name())
}

// UnsyncableNamespaceObjectError represents an illegal usage of a namespace object kind which has not been explicitly declared.
type UnsyncableNamespaceObjectError struct {
	*ast.NamespaceObject
}

// Error implements error.
func (e UnsyncableNamespaceObjectError) Error() string {
	return fmt.Sprintf(
		"KNV1006: Object %[4]q is not syncable. "+
			"Enable sync for objects of kind %[2]s.\n"+
			"%[3]s",
		repo.ClusterDir, e.GroupVersionKind(), e.FileObject, e.Name())
}

// IllegalAbstractNamespaceObjectKindError represents an illegal usage of a kind not allowed in abstract namespaces.
type IllegalAbstractNamespaceObjectKindError struct {
	*ast.NamespaceObject
}

// Error implements error.
func (e IllegalAbstractNamespaceObjectKindError) Error() string {
	return fmt.Sprintf(
		"KNV1007: Objects of kind %[1]s MUST NOT be delcared in %[2]s directories. \n"+
			"Move object %[5]q to a %[3]s directory:\n"+
			"%[4]s",
		e.GroupVersionKind(), ast.AbstractNamespace, ast.Namespace, e.FileObject, e.Name())
}

// ConflictingResourceQuotaError represents multiple ResourceQuotas illegally presiding in the same directory.
type ConflictingResourceQuotaError struct {
	*ast.NamespaceObject
}

// Error implements error.
func (e ConflictingResourceQuotaError) Error() string {
	return fmt.Sprintf(
		"KNV1008: A directory MUST NOT contain more than one ResourceQuota object. "+
			"Directory %[1]q contains multiple ResourceQuota object definitions, including:\n"+
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
	return fmt.Sprintf(
		"KNV1009: Objects MUST NOT delcare metadata.namespace. "+
			"Object %[2]q declares metadata.namespace:\n"+
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
	a := strings.Join(e.annotations, ", ")
	return fmt.Sprintf(
		"KNV1010: Objects MUST NOT define unsupported annotations starting with %[3]q. "+
			"Object %[4]q has offending annotations: %[1]s\n%[2]s",
		a, e.object, policyhierarchy.GroupName, e.object.Name())
}

// IllegalLabelDefinitionError represent a set of illegal label definitions.
type IllegalLabelDefinitionError struct {
	object ast.FileObject
	labels []string
}

// Error implements error.
func (e IllegalLabelDefinitionError) Error() string {
	l := strings.Join(e.labels, ", ")
	return fmt.Sprintf(
		"KNV1011: Objects MUST NOT define labels starting with %[3]q. "+
			"Object  %[3]s has these offending labels: %[1]s\n%[2]s",
		l, e.object, policyhierarchy.GroupName, e.object.Name())
}

// NamespaceSelectorMayNotHaveAnnotation reports that a namespace selector has
// an annotation that is not allowed.
type NamespaceSelectorMayNotHaveAnnotation struct {
	o metav1.Object
}

// Error implements error.
func (e NamespaceSelectorMayNotHaveAnnotation) Error() string {
	return fmt.Sprintf("KNV1012: The NamespaceSelector object %q in namespace %q must not have ClusterSelector annotation", e.o.GetName(), e.o.GetNamespace())
}
