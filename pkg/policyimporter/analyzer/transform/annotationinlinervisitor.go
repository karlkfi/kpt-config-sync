package transform

import (
	"encoding/json"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
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
	// cv is used for copying parts of the ast.Context tree and continuing underlying visitor iteration.
	cv *visitor.Copying
	// transformer is created and set for each TreeNode
	transformer annotationTransformer
	// cumulative errors encountered by the visitor
	errs *multierror.Builder
}

var _ ast.Visitor = &AnnotationInlinerVisitor{}

// NewAnnotationInlinerVisitor returns a new AnnotationInlinerVisitor
func NewAnnotationInlinerVisitor() *AnnotationInlinerVisitor {
	cv := visitor.NewCopying()
	v := &AnnotationInlinerVisitor{
		cv:   cv,
		errs: multierror.NewBuilder(),
	}
	cv.SetImpl(v)
	return v
}

// Result implements MutatingVisitor
func (v *AnnotationInlinerVisitor) Result() error {
	return v.errs.Build()
}

// VisitContext implements Visitor
func (v *AnnotationInlinerVisitor) VisitContext(g *ast.Context) ast.Node {
	return v.cv.VisitContext(g)
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
		if err := validateSelector(s); err != nil {
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
	v.transformer.addMappingForKey(v1.NamespaceSelectorAnnotationKey, m)
	return v.cv.VisitTreeNode(n).(*ast.TreeNode)
}

// VisitObjectList implements Visitor
func (v *AnnotationInlinerVisitor) VisitObjectList(o ast.ObjectList) ast.Node {
	return v.cv.VisitObjectList(o)
}

// VisitObject implements Visitor
func (v *AnnotationInlinerVisitor) VisitObject(o *ast.Object) ast.Node {
	newObject := v.cv.VisitObject(o).(*ast.Object)
	if err := v.transformer.transform(newObject.ToMeta()); err != nil {
		v.errs.Add(errors.Wrapf(err, "failed to inline annotation for object %q", newObject.ToMeta().GetName()))
	}
	return newObject
}
