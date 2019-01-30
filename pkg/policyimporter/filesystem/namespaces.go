package filesystem

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/node"
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
	errorBuilder *multierror.Builder) {
	metadata.Validate(toResourceMetas(objects), errorBuilder)

	syntax.DirectoryNameValidator.Validate(dirs, errorBuilder)
	syntax.DisallowSystemObjectsValidator.Validate(objects, errorBuilder)

	semantic.DuplicateDirectoryValidator{Dirs: dirs}.Validate(errorBuilder)
	semantic.DuplicateNamespaceValidator{Objects: objects}.Validate(errorBuilder)
}

func processNamespaces(
	dir nomospath.Relative,
	objects []ast.FileObject,
	treeGenerator *DirectoryTree) {
	treeNode := treeGenerator.AddDir(dir)
	// TODO: Put this in a transforming Visitor.
	for _, object := range objects {
		switch o := object.Object.(type) {
		case *corev1.Namespace:
			treeNode.Type = node.Namespace
			metaObj := object.Object.(metav1.Object)
			treeNode.Labels = metaObj.GetLabels()
			treeNode.Annotations = metaObj.GetAnnotations()
		case *v1alpha1.NamespaceSelector:
			treeNode.Selectors[o.Name] = o
		default:
			treeNode.Objects = append(treeNode.Objects, &ast.NamespaceObject{FileObject: object})
		}
	}
}
