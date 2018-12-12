package filesystem

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/semantic"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/util/multierror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// validateNamespaces validates all Resources in namespaces/ including Namespaces and Abstract
// Namespaces.
func validateNamespaces(objects []ast.FileObject, dirs []string, errorBuilder *multierror.Builder) {
	validateObjects(objects, errorBuilder)

	syntax.DirectoryNameValidator.Validate(dirs, errorBuilder)
	syntax.DisallowSystemObjectsValidator.Validate(objects, errorBuilder)

	semantic.ConflictingResourceQuotaValidator{Objects: objects}.Validate(errorBuilder)
	semantic.DuplicateDirectoryValidator{Dirs: dirs}.Validate(errorBuilder)
	semantic.DuplicateNamespaceValidator{Objects: objects}.Validate(errorBuilder)
}

func processNamespaces(
	dir string,
	objects []ast.FileObject,
	namespaceDirs map[string]bool,
	treeGenerator *DirectoryTree,
	root bool, errorBuilder *multierror.Builder) {
	var treeNode *ast.TreeNode
	for _, object := range objects {
		switch object.Object.(type) {
		case *corev1.Namespace:
			namespaceDirs[dir] = true
			if root {
				treeNode = treeGenerator.SetRootDir(dir, ast.Namespace)
			} else {
				treeNode = treeGenerator.AddDir(dir, ast.Namespace)
			}
			processNamespace(objects, treeNode, errorBuilder)
			return
		}
	}
	// No namespace resource was found.
	if root {
		treeNode = treeGenerator.SetRootDir(dir, ast.AbstractNamespace)
	} else {
		treeNode = treeGenerator.AddDir(dir, ast.AbstractNamespace)
	}

	for _, i := range objects {
		switch o := i.Object.(type) {
		case *v1alpha1.NamespaceSelector:
			treeNode.Selectors[o.Name] = o
		default:
			treeNode.Objects = append(treeNode.Objects, &ast.NamespaceObject{FileObject: ast.FileObject{Object: o, Source: i.Source}})
		}
	}
}

func processNamespace(objects []ast.FileObject, treeNode *ast.TreeNode, errorBuilder *multierror.Builder) {
	validateNamespace(objects, errorBuilder)

	for _, object := range objects {
		gvk := object.GroupVersionKind()
		if gvk == corev1.SchemeGroupVersion.WithKind("Namespace") {
			// TODO: Move this out.
			metaObj := object.Object.(metav1.Object)
			treeNode.Labels = metaObj.GetLabels()
			treeNode.Annotations = metaObj.GetAnnotations()
			continue
		}
		treeNode.Objects = append(treeNode.Objects, &ast.NamespaceObject{FileObject: ast.FileObject{Object: object.Object, Source: object.Source}})
	}
}

// Validation specific to Namespaces. These validations do not apply to Abstract Namespaces.
func validateNamespace(objects []ast.FileObject, errorBuilder *multierror.Builder) {
	syntax.NamespacesKindValidator.Validate(objects, errorBuilder)
}
