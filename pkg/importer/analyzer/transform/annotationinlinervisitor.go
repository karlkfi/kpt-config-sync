package transform

import (
	"encoding/json"

	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	sel "github.com/google/nomos/pkg/importer/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
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
	// Used to inline cluster selector annotations.  It is created anew for each traversal.
	clusterSelectorTransformer annotationTransformer
	// selectors contains the cluster selection data.
	selectors *sel.ClusterSelectors
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

// VisitRoot implements ast.Visitor.
func (v *AnnotationInlinerVisitor) VisitRoot(r *ast.Root) *ast.Root {
	glog.V(5).Infof("VisitRoot(): ENTER")
	defer glog.V(6).Infof("VisitRoot(): EXIT")
	cs, err := sel.GetClusterSelectors(r)
	v.errs = status.Append(v.errs, err)
	v.selectors = cs
	// Add inliner map for cluster annotations.
	t := annotationTransformer{}
	m := valueMap{}
	cs.ForEachSelector(func(name string, annotation v1.ClusterSelector) {
		content, err := json.Marshal(annotation)
		if err != nil {
			// TODO(b/122739070) ast.Root should store the ClusterSelectors rather than having to transform them every time.
			v.errs = status.Append(v.errs, status.UndocumentedWrapf(err, "failed to marshal ClusterSelector %q", name))
			return
		}
		m[name] = string(content)
	})
	t.addMappingForKey(v1.ClusterSelectorAnnotationKey, m)
	v.clusterSelectorTransformer = t
	return v.Copying.VisitRoot(r)
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
			v.errs = status.Append(v.errs, vet.InvalidSelectorError(s.Name, err))
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

	v.errs = status.Append(v.errs, status.UndocumentedWrapf(v.clusterSelectorTransformer.transform(n), "failed to inline ClusterSelector for node %q", n.SlashPath()))
	setPopulatedAnnotation(n, v1.ClusterNameAnnotationKey, v.selectors.ClusterName())
	return v.Copying.VisitTreeNode(n)
}

// VisitObject implements Visitor
func (v *AnnotationInlinerVisitor) VisitObject(o *ast.NamespaceObject) *ast.NamespaceObject {
	glog.V(5).Infof("VisitObject(): ENTER")
	defer glog.V(6).Infof("VisitObject(): EXIT")
	newObject := v.Copying.VisitObject(o)
	v.errs = status.Append(v.errs, status.UndocumentedWrapf(v.nsTransformer.transform(newObject),
		"failed to inline annotation for object %q", newObject.GetName()))
	v.errs = status.Append(v.errs, status.UndocumentedWrapf(v.clusterSelectorTransformer.transform(newObject),
		"failed to inline cluster selector annotations for object %q", newObject.GetName()))
	setPopulatedAnnotation(newObject, v1.ClusterNameAnnotationKey, v.selectors.ClusterName())
	return newObject
}

// VisitClusterObject implements Visitor
func (v *AnnotationInlinerVisitor) VisitClusterObject(o *ast.ClusterObject) *ast.ClusterObject {
	glog.V(5).Infof("VisitClusterObject(): ENTER")
	defer glog.V(6).Infof("VisitClusterObject(): EXIT")
	newObject := v.Copying.VisitClusterObject(o)
	v.errs = status.Append(v.errs, status.InternalWrapf(v.clusterSelectorTransformer.transform(newObject),
		"failed to inline cluster selector annotations for object %q", newObject.GetName()))
	setPopulatedAnnotation(newObject, v1.ClusterNameAnnotationKey, v.selectors.ClusterName())
	return newObject
}

// setPopulatedAnnotation is like object.SetAnnotation, but only populates the annotation if value
// is not the empty string.
func setPopulatedAnnotation(obj core.Annotated, annotation, value string) {
	if value == "" {
		return
	}
	core.SetAnnotation(obj, annotation, value)
}
