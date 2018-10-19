package transform

import (
	"encoding/json"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/util/multierror"
	"github.com/pkg/errors"
)

// AnnotationInlinerVisitor inlines annotation values.
//
// For example, the following annotation:
//
// nomos.dev/namespace-selector: sre-supported
//
// Would be inlined to:
//
// nomos.dev/namespace-selector: {\"kind\": \"NamespaceSelector\",..}
type AnnotationInlinerVisitor struct {
	// Copying is used for copying parts of the ast.Root tree and continuing underlying visitor iteration.
	*visitor.Copying
	// transformer is created and set for each TreeNode
	transformer annotationTransformer
	// cumulative errors encountered by the visitor
	errs *multierror.Builder
}

var _ ast.Visitor = &AnnotationInlinerVisitor{}

// NewAnnotationInlinerVisitor returns a new AnnotationInlinerVisitor
func NewAnnotationInlinerVisitor() *AnnotationInlinerVisitor {
	v := &AnnotationInlinerVisitor{
		Copying: visitor.NewCopying(),
		errs:    multierror.NewBuilder(),
	}
	v.SetImpl(v)
	return v
}

// Error implements CheckingVisitor
func (v *AnnotationInlinerVisitor) Error() error {
	return v.errs.Build()
}

// VisitReservedNamespaces implements Visitor
func (v *AnnotationInlinerVisitor) VisitReservedNamespaces(r *ast.ReservedNamespaces) ast.Node {
	return r
}

// VisitCluster implements Visitor
func (v *AnnotationInlinerVisitor) VisitCluster(c *ast.Cluster) ast.Node {
	return c
}

// VisitTreeNode implements Visitor
func (v *AnnotationInlinerVisitor) VisitTreeNode(n *ast.TreeNode) ast.Node {
	m := valueMap{}
	for k, s := range n.Selectors {
		if n.Type == ast.Namespace {
			// This should already be validated in parser.
			v.errs.Add(errors.Errorf("NamespaceSelector must not be in namespace directories, found in %q", n.Path))
			return n
		}
		if _, err := asPopulatedSelector(&s.Spec.Selector); err != nil {
			v.errs.Add(errors.Wrapf(err, "NamespaceSelector %q is not valid", s.Name))
			continue
		}
		content, err := json.Marshal(s)
		if err != nil {
			// This should already be validated in parser.
			v.errs.Add(errors.Wrapf(err, "failed to marshal NamespaceSelector %q", s.Name))
			continue
		}
		m[k] = string(content)
	}
	v.transformer = annotationTransformer{}
	v.transformer.addMappingForKey(v1alpha1.NamespaceSelectorAnnotationKey, m)
	return v.Copying.VisitTreeNode(n).(*ast.TreeNode)
}

// VisitObject implements Visitor
func (v *AnnotationInlinerVisitor) VisitObject(o *ast.NamespaceObject) ast.Node {
	newObject := v.Copying.VisitObject(o).(*ast.NamespaceObject)
	if err := v.transformer.transform(newObject.ToMeta()); err != nil {
		v.errs.Add(errors.Wrapf(err, "failed to inline annotation for object %q", newObject.ToMeta().GetName()))
	}
	return newObject
}
