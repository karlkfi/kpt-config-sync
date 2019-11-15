package validation

import (
	"strings"

	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/analyzer/transform"
	"github.com/google/nomos/pkg/importer/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/importer/analyzer/validation/coverage"
	"github.com/google/nomos/pkg/importer/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// InputValidator checks various filesystem constraints after loading into the tree format.
// Error messages emitted from the validator should be formatted to first print the constraint
// that is being violated, then print a useful error message on what is violating the constraint
// and what is required to fix it.
type InputValidator struct {
	*visitor.Base
	errs             status.MultiError
	nodes            []*ast.TreeNode
	coverage         *coverage.ForCluster
	inheritanceSpecs map[schema.GroupKind]*transform.InheritanceSpec
}

// InputValidator implements ast.Visitor
var _ ast.Visitor = &InputValidator{}

// NewInputValidator creates a new validator.  syncdGVKs represents the set
// of valid group-version-kinds for objects in the namespaces and cluster
// directories.  Objects of other types will be treated as an error. clusters
// is the list of clusters defined in the source of truth, and cs is the list
// of selectors.  vet turns on "vetting mode", a mode of stricter control for use
// in nomos vet.
func NewInputValidator(specs map[schema.GroupKind]*transform.InheritanceSpec) *InputValidator {
	v := &InputValidator{
		Base:             visitor.NewBase(),
		inheritanceSpecs: specs,
	}
	v.Base.SetImpl(v)

	return v
}

// Error returns any errors encountered during processing
func (v *InputValidator) Error() status.MultiError {
	return v.errs
}

// VisitRoot gets the clusters and selectors stored in Root.Data and constructs coverage if vet is
// enabled.
func (v *InputValidator) VisitRoot(r *ast.Root) *ast.Root {
	clusters, err := selectors.GetClusters(r)
	v.errs = status.Append(v.errs, err)
	sels, err := selectors.GetSelectors(r)
	v.errs = status.Append(v.errs, err)
	v.coverage, v.errs = coverage.NewForCluster(clusters, sels)

	return v.Base.VisitRoot(r)
}

// VisitTreeNode implements Visitor
func (v *InputValidator) VisitTreeNode(n *ast.TreeNode) *ast.TreeNode {
	// Namespaces may not have children.
	if len(v.nodes) > 1 {
		// Recall that v.nodes are this node's ancestors in the tree of directories.
		// If len == 0, this node has no ancestors and so cannot be the child of a Namespace directory.
		// If len == 1, this is a child of namespaces/ and so it cannot be the child of a Namespace directory.
		// We check for the two cases above elsewhere, so adding errors here adds noise and incorrect advice.
		if parent := v.nodes[len(v.nodes)-1]; parent.Type == node.Namespace {
			v.errs = status.Append(v.errs, IllegalNamespaceSubdirectoryError(n, parent))
		}
	}
	for _, s := range n.Selectors {
		v.checkNamespaceSelectorAnnotations(s)
	}

	v.nodes = append(v.nodes, n)
	o := v.Base.VisitTreeNode(n)
	v.nodes = v.nodes[:len(v.nodes)-1]
	// Must return non-nil so that visiting may continue to cluster objects.
	return o
}

// checkNamespaceSelectorAnnotations ensures that a NamespaceSelector object has no
// ClusterSelector annotation on it.
func (v *InputValidator) checkNamespaceSelectorAnnotations(s *v1.NamespaceSelector) {
	if a := s.GetAnnotations(); a != nil {
		if _, ok := a[v1.ClusterSelectorAnnotationKey]; ok {
			v.errs = status.Append(v.errs, NamespaceSelectorMayNotHaveAnnotation(s))
		}
	}
}

// VisitClusterObject implements Visitor
func (v *InputValidator) VisitClusterObject(o *ast.ClusterObject) *ast.ClusterObject {
	if v.coverage != nil {
		v.errs = status.Append(v.errs, v.coverage.ValidateObject(&o.FileObject))
	}
	return v.Base.VisitClusterObject(o)
}

// VisitObject implements Visitor
func (v *InputValidator) VisitObject(o *ast.NamespaceObject) *ast.NamespaceObject {
	// TODO: Move each individual check here to its own Visitor.
	gvk := o.GroupVersionKind()

	n := v.nodes[len(v.nodes)-1]
	if n.Type == node.AbstractNamespace {
		spec, found := v.inheritanceSpecs[gvk.GroupKind()]
		if (found && spec.Mode == v1.HierarchyModeNone) && !transform.IsEphemeral(gvk) && !syntax.IsSystemOnly(gvk) {
			v.errs = status.Append(v.errs, IllegalAbstractNamespaceObjectKindError(o))
		}
	}

	if v.coverage != nil {
		v.errs = status.Append(v.errs, v.coverage.ValidateObject(&o.FileObject))
	}

	return v.Base.VisitObject(o)
}

// IllegalNamespaceSubdirectoryErrorCode is the error code for IllegalNamespaceSubdirectoryError
const IllegalNamespaceSubdirectoryErrorCode = "1003"

var illegalNamespaceSubdirectoryError = status.NewErrorBuilder(IllegalNamespaceSubdirectoryErrorCode)

// IllegalNamespaceSubdirectoryError represents an illegal child directory of a namespace directory.
func IllegalNamespaceSubdirectoryError(child, parent id.TreeNode) status.Error {
	// TODO: We don't really need the parent node since it can be inferred from the Child.
	return illegalNamespaceSubdirectoryError.Sprintf("A %[1]s directory MUST NOT have subdirectories. "+
		"Restructure %[4]q so that it does not have subdirectory %[2]q:\n\n"+
		"%[3]s",
		node.Namespace, child.Name(), id.PrintTreeNode(child), parent.Name()).BuildWithPaths(child, parent)
}

// IllegalAbstractNamespaceObjectKindErrorCode is the error code for IllegalAbstractNamespaceObjectKindError
const IllegalAbstractNamespaceObjectKindErrorCode = "1007"

var illegalAbstractNamespaceObjectKindError = status.NewErrorBuilder(IllegalAbstractNamespaceObjectKindErrorCode)

// IllegalAbstractNamespaceObjectKindError represents an illegal usage of a kind not allowed in abstract namespaces.
// TODO(willbeason): Consolidate Illegal{X}ObjectKindErrors
func IllegalAbstractNamespaceObjectKindError(resource id.Resource) status.Error {
	return illegalAbstractNamespaceObjectKindError.Sprintf(
		"Config `%[3]s` illegally declared in an %[1]s directory. "+
			"Move this config to a %[2]s directory:",
		strings.ToLower(string(node.AbstractNamespace)), node.Namespace, resource.GetName()).Build()
}

// NamespaceSelectorMayNotHaveAnnotationCode is the error code for NamespaceSelectorMayNotHaveAnnotation
const NamespaceSelectorMayNotHaveAnnotationCode = "1012"

var namespaceSelectorMayNotHaveAnnotation = status.NewErrorBuilder(NamespaceSelectorMayNotHaveAnnotationCode)

// NamespaceSelectorMayNotHaveAnnotation reports that a namespace selector has
// an annotation that is not allowed.
func NamespaceSelectorMayNotHaveAnnotation(object core.Object) status.Error {
	// TODO(willbeason): Print information about the object so it can actually be found.
	return namespaceSelectorMayNotHaveAnnotation.Sprintf("The NamespaceSelector config %q MUST NOT have ClusterSelector annotation", object.GetName()).Build()
}
