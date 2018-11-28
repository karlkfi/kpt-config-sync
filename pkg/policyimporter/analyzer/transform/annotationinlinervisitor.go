/*
Copyright 2018 The Nomos Authors.

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

package transform

import (
	"encoding/json"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	sel "github.com/google/nomos/pkg/policyimporter/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/util/multierror"
	"github.com/pkg/errors"
)

// AnnotationInlinerVisitor inlines annotation values. Inlining replaces the
// annotation value with the verbatim JSON-formatted content of a Selector that
// matches the annotation value.
//
// Replaces the following annotations:
// - nomos.dev/namespace-selector: sre-supported
// - nomos.dev/cluster-selector: production
//
// nomos.dev/namespace-selector: sre-supported
//
// Would be inlined to:
//
// nomos.dev/namespace-selector: {\"kind\": \"NamespaceSelector\",..}
// where the replacement is the NamespaceSelector named "sre-supported", in
// JSON format.
type AnnotationInlinerVisitor struct {
	// Copying is used for copying parts of the ast.Root tree and continuing underlying visitor iteration.
	*visitor.Copying
	// nsTransformer is used to inline namespace selector annotations. It is
	// created anew for each TreeNode.
	nsTransformer annotationTransformer
	// cumulative errors encountered by the visitor
	errs multierror.Builder
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

// Error implements CheckingVisitor
func (v *AnnotationInlinerVisitor) Error() error {
	return v.errs.Build()
}

// VisitRoot implements ast.Visitor.
func (v *AnnotationInlinerVisitor) VisitRoot(r *ast.Root) *ast.Root {
	glog.V(5).Infof("VisitRoot(): ENTER")
	defer glog.V(6).Infof("VisitRoot(): EXIT")
	cs := sel.GetClusterSelectors(r)
	v.selectors = cs
	// Add inliner map for cluster annotations.
	t := annotationTransformer{}
	m := valueMap{}
	cs.ForEachSelector(func(name string, annotation v1alpha1.ClusterSelector) {
		content, err := json.Marshal(annotation)
		if err != nil {
			// This should already be validated in parser.
			v.errs.Add(errors.Wrapf(err, "failed to marshal ClusterSelector %q", name))
			return
		}
		m[name] = string(content)
	})
	t.addMappingForKey(v1alpha1.ClusterSelectorAnnotationKey, m)
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
		if n.Type == ast.Namespace {
			// This should already be validated in parser.
			v.errs.Add(errors.Errorf("NamespaceSelector must not be in namespace directories, found in %q", n.Path))
			return n
		}
		if _, err := sel.AsPopulatedSelector(&s.Spec.Selector); err != nil {
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
	v.nsTransformer = annotationTransformer{}
	v.nsTransformer.addMappingForKey(v1alpha1.NamespaceSelectorAnnotationKey, m)

	v.errs.Add(errors.Wrapf(v.clusterSelectorTransformer.transform(n), "failed to inline ClusterSelector for node %q", n.Path))
	annotatePopulated(n, v1alpha1.ClusterNameAnnotationKey, v.selectors.ClusterName())
	return v.Copying.VisitTreeNode(n)
}

// VisitObject implements Visitor
func (v *AnnotationInlinerVisitor) VisitObject(o *ast.NamespaceObject) *ast.NamespaceObject {
	glog.V(5).Infof("VisitObject(): ENTER")
	defer glog.V(6).Infof("VisitObject(): EXIT")
	newObject := v.Copying.VisitObject(o)
	m := newObject.ToMeta()
	v.errs.Add(errors.Wrapf(v.nsTransformer.transform(m),
		"failed to inline annotation for object %q", m.GetName()))
	v.errs.Add(errors.Wrapf(v.clusterSelectorTransformer.transform(m),
		"failed to inline cluster selector annotations for object %q", m.GetName()))
	annotatePopulated(m, v1alpha1.ClusterNameAnnotationKey, v.selectors.ClusterName())
	return newObject
}

// VisitClusterObject implements Visitor
func (v *AnnotationInlinerVisitor) VisitClusterObject(o *ast.ClusterObject) *ast.ClusterObject {
	glog.V(5).Infof("VisitClusterObject(): ENTER")
	defer glog.V(6).Infof("VisitClusterObject(): EXIT")
	newObject := o.DeepCopy()
	m := newObject.ToMeta()
	v.errs.Add(errors.Wrapf(v.clusterSelectorTransformer.transform(m),
		"failed to inline cluster selector annotations for object %q", m.GetName()))
	annotatePopulated(m, v1alpha1.ClusterNameAnnotationKey, v.selectors.ClusterName())
	return newObject
}
