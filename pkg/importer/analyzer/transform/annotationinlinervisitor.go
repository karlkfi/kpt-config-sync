package transform

import (
	"encoding/json"

	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	sel "github.com/google/nomos/pkg/importer/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/status"
)

// AnnotationInlinerVisitor inlines annotation values. Inlining replaces the
// annotation value with the verbatim JSON-formatted content of a Selector that
// matches the annotation value.
//
// Replaces the following annotations:
// - configmanagement.gke.io/namespace-selector: sre-supported
// - configmanagement.gke.io/cluster-selector: production
//
// configmanagement.gke.io/namespace-selector: sre-supported
//
// Would be inlined to:
//
// configmanagement.gke.io/namespace-selector: {\"kind\": \"NamespaceSelector\",..}
// where the replacement is the NamespaceSelector named "sre-supported", in
// JSON format.
type AnnotationInlinerVisitor struct {
	// Copying is used for copying parts of the ast.Root tree and continuing underlying visitor iteration.
	*visitor.Copying
	// nsTransformer is used to inline namespace selector annotations. It is
	// created anew for each TreeNode.
	nsTransformer annotationTransformer
	// cumulative errors encountered by the visitor
	errs status.MultiError
}

var _ ast.Visitor = &AnnotationInlinerVisitor{}

// NewAnnotationInlinerVisitor returns a new AnnotationInlinerVisitor. cs is the
// cluster selector to use for inlining.
func NewAnnotationInlinerVisitor() *AnnotationInlinerVisitor {
	v := &AnnotationInlinerVisitor{
		Copying: visitor.NewCopying(),
	}
	v.SetImpl(v)
	return v
}

// Error implements Visitor
func (v *AnnotationInlinerVisitor) Error() status.MultiError {
	return v.errs
}

// VisitTreeNode implements Visitor
func (v *AnnotationInlinerVisitor) VisitTreeNode(n *ast.TreeNode) *ast.TreeNode {
	glog.V(5).Infof("VisitTreeNode(): ENTER")
	defer glog.V(6).Infof("VisitTreeNode(): EXIT")
	n = n.PartialCopy()
	m := valueMap{}
	for k, s := range n.Selectors {
		if n.Type == node.Namespace {
			// TODO(b/122739070) This should already be validated in parser.
			v.errs = status.Append(v.errs, status.UndocumentedErrorf("NamespaceSelector must not be in namespace directories, found in %q", n.SlashPath()))
			return n
		}
		if _, err := sel.AsPopulatedSelector(&s.Spec.Selector); err != nil {
			// TODO(b/122739070) This should already be validated in parser.
			v.errs = status.Append(v.errs, sel.InvalidSelectorError(s.Name, err))
			continue
		}
		content, err := json.Marshal(s)
		if err != nil {
			// TODO(b/122739070) This should already be validated in parser.
			v.errs = status.Append(v.errs, status.UndocumentedWrapf(err, "failed to marshal NamespaceSelector %q", s.Name))
			continue
		}
		m[k] = string(content)
	}
	v.nsTransformer = annotationTransformer{}
	v.nsTransformer.addMappingForKey(v1.NamespaceSelectorAnnotationKey, m)
	return v.Copying.VisitTreeNode(n)
}

// VisitObject implements Visitor
func (v *AnnotationInlinerVisitor) VisitObject(o *ast.NamespaceObject) *ast.NamespaceObject {
	glog.V(5).Infof("VisitObject(): ENTER")
	defer glog.V(6).Infof("VisitObject(): EXIT")
	newObject := v.Copying.VisitObject(o)
	v.errs = status.Append(v.errs, status.UndocumentedWrapf(v.nsTransformer.transform(newObject),
		"failed to inline annotation for object %q", newObject.GetName()))
	return newObject
}
