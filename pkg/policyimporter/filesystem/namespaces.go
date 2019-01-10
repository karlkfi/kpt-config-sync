package filesystem

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/node"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/coverage"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/metadata"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/semantic"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	"github.com/google/nomos/pkg/util/multierror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// validateNamespaces validates all Resources in namespaces/ including Namespaces and Abstract
// Namespaces.
func validateNamespaces(
	objects []ast.FileObject,
	dirs []nomospath.Relative,
	cov *coverage.ForCluster,
	errorBuilder *multierror.Builder) {
	metadata.Validate(toResourceMetas(objects), errorBuilder)

	syntax.DirectoryNameValidator.Validate(dirs, errorBuilder)
	syntax.DisallowSystemObjectsValidator.Validate(objects, errorBuilder)

	semantic.NewConflictingResourceQuotaValidator(objects, cov).Validate(errorBuilder)
	semantic.DuplicateDirectoryValidator{Dirs: dirs}.Validate(errorBuilder)
	semantic.DuplicateNamespaceValidator{Objects: objects}.Validate(errorBuilder)
}

func processNamespaces(
	dir nomospath.Relative,
	objects []ast.FileObject,
	treeGenerator *DirectoryTree,
	errorBuilder *multierror.Builder) {
	for _, object := range objects {
		switch object.Object.(type) {
		case *corev1.Namespace:
			treeNode := treeGenerator.AddDir(dir, node.Namespace)
			processNamespace(objects, treeNode, errorBuilder)
			return
		}
	}
	// No namespace resource was found.
	treeNode := treeGenerator.AddDir(dir, node.AbstractNamespace)

	for _, i := range objects {
		switch o := i.Object.(type) {
		case *v1alpha1.NamespaceSelector:
			treeNode.Selectors[o.Name] = o
		default:
			treeNode.Objects = append(treeNode.Objects, &ast.NamespaceObject{FileObject: ast.NewFileObject(o, i.Relative)})
		}
	}
}

func processNamespace(objects []ast.FileObject, treeNode *ast.TreeNode, errorBuilder *multierror.Builder) {
	validateNamespace(objects, errorBuilder)

	for _, object := range objects {
		gvk := object.GroupVersionKind()
		if gvk == kinds.Namespace() {
			// TODO: Move this out.
			metaObj := object.Object.(metav1.Object)
			treeNode.Labels = metaObj.GetLabels()
			treeNode.Annotations = metaObj.GetAnnotations()
			continue
		}
		treeNode.Objects = append(treeNode.Objects, &ast.NamespaceObject{FileObject: ast.NewFileObject(object.Object, object.Relative)})
	}
}

// Validation specific to Namespaces. These validations do not apply to Abstract Namespaces.
func validateNamespace(objects []ast.FileObject, errorBuilder *multierror.Builder) {
	syntax.NamespacesKindValidator.Validate(objects, errorBuilder)
}
