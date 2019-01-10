/*
Copyright 2017 The Nomos Authors.
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

package validation

import (
	"path"

	"github.com/google/nomos/pkg/util/namespaceutil"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/transform"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/coverage"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/util/multierror"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

// InputValidator checks various filesystem constraints after loading into the tree format.
// Error messages emitted from the validator should be formatted to first print the constraint
// that is being violated, then print a useful error message on what is violating the constraint
// and what is required to fix it.
type InputValidator struct {
	*visitor.Base
	errs             multierror.Builder
	nodes            []*ast.TreeNode
	allowedGVKs      map[schema.GroupVersionKind]bool
	coverage         *coverage.ForCluster
	inheritanceSpecs map[schema.GroupKind]*transform.InheritanceSpec
}

// InputValidator implements ast.Visitor
var _ ast.Visitor = &InputValidator{}

// NewInputValidator creates a new validator.  allowedGVKs represents the set
// of valid group-version-kinds for objects in the namespaces and cluster
// directories.  Objects of other types will be treated as an error. clusters
// is the list of clusters defined in the source of truth, and cs is the list
// of selectors.  vet turns on "vetting mode", a mode of stricter control for use
// in nomos vet.
func NewInputValidator(
	syncs []*v1alpha1.Sync,
	specs map[schema.GroupKind]*transform.InheritanceSpec,
	clusters []clusterregistry.Cluster,
	cs []v1alpha1.ClusterSelector,
	vet bool) *InputValidator {
	v := &InputValidator{
		Base:             visitor.NewBase(),
		allowedGVKs:      toAllowedGVKs(syncs),
		inheritanceSpecs: specs,
	}
	v.Base.SetImpl(v)

	if vet {
		v.coverage = coverage.NewForCluster(clusters, cs, &v.errs)
	}
	return v
}

func toAllowedGVKs(syncs []*v1alpha1.Sync) map[schema.GroupVersionKind]bool {
	allowedGVKs := make(map[schema.GroupVersionKind]bool)
	for _, sync := range syncs {
		for _, sg := range sync.Spec.Groups {
			for _, k := range sg.Kinds {
				for _, v := range k.Versions {
					gvk := schema.GroupVersionKind{Group: sg.Group, Kind: k.Kind, Version: v.Version}
					allowedGVKs[gvk] = true
				}
			}
		}
	}
	return allowedGVKs
}

// Error returns any errors encountered during processing
func (v *InputValidator) Error() error {
	return v.errs.Build()
}

// VisitTreeNode implements Visitor
func (v *InputValidator) VisitTreeNode(n *ast.TreeNode) *ast.TreeNode {
	name := path.Base(n.Path)
	if namespaceutil.IsReserved(name) {
		// The node's name must not be a reserved namespace name.
		v.errs.Add(veterrors.ReservedDirectoryNameError{Dir: n.Path})
	}

	// Namespaces may not have children.
	if len(v.nodes) > 1 {
		// Recall that v.nodes are this node's ancestors in the tree of directories.
		// If len == 0, this node has no ancestors and so cannot be the child of a Namespace directory.
		// If len == 1, this is a child of namespaces/ and so it cannot be the child of a Namespace directory.
		// We check for the two cases above elsewhere, so adding errors here adds noise and incorrect advice.
		if parent := v.nodes[len(v.nodes)-1]; parent.Type == ast.Namespace {
			v.errs.Add(veterrors.IllegalNamespaceSubdirectoryError{Child: n, Parent: parent})
		}
	}
	for _, s := range n.Selectors {
		v.checkNamespaceSelectorAnnotations(s)
	}

	if n.Type == ast.Namespace {
		// Namespace-specific validation
		if _, found := n.Annotations[v1alpha1.NamespaceSelectorAnnotationKey]; found {
			// Namespaces may not use the selector annotation.
			v.errs.Add(veterrors.IllegalNamespaceSelectorAnnotationError{TreeNode: n})
		}
	}

	v.nodes = append(v.nodes, n)
	o := v.Base.VisitTreeNode(n)
	v.nodes = v.nodes[:len(v.nodes)-1]
	// Must return non-nil so that visiting may continue to cluster objects.
	return o
}

// checkNamespaceSelectorAnnotations ensures that a NamespaceSelector object has no
// ClusterSelector annotation on it.
func (v *InputValidator) checkNamespaceSelectorAnnotations(s *v1alpha1.NamespaceSelector) {
	if a := s.GetAnnotations(); a != nil {
		if _, ok := a[v1alpha1.ClusterSelectorAnnotationKey]; ok {
			v.errs.Add(veterrors.NamespaceSelectorMayNotHaveAnnotation{Object: s})
		}
	}
}

// VisitClusterObject implements Visitor
func (v *InputValidator) VisitClusterObject(o *ast.ClusterObject) *ast.ClusterObject {
	gvk := o.GroupVersionKind()
	if !v.allowedGVKs[gvk] {
		v.errs.Add(veterrors.UnsyncableClusterObjectError{ResourceID: o})
	}
	if v.coverage != nil {
		v.coverage.ValidateObject(o.MetaObject(), &v.errs)
	}
	return v.Base.VisitClusterObject(o)
}

// VisitObject implements Visitor
func (v *InputValidator) VisitObject(o *ast.NamespaceObject) *ast.NamespaceObject {
	if !v.allowedGVKs[o.GroupVersionKind()] {
		if !syntax.IsSystemOnly(o.GroupVersionKind()) {
			// This is already checked elsewhere.
			v.errs.Add(veterrors.UnsyncableNamespaceObjectError{ResourceID: o})
		}
	}

	node := v.nodes[len(v.nodes)-1]
	if node.Type == ast.AbstractNamespace {
		spec, found := v.inheritanceSpecs[o.GroupVersionKind().GroupKind()]
		if !found || spec.Mode == v1alpha1.HierarchyModeNone {
			v.errs.Add(veterrors.IllegalAbstractNamespaceObjectKindError{ResourceID: o})
		}
	}

	if v.coverage != nil {
		v.coverage.ValidateObject(o.MetaObject(), &v.errs)
	}

	return v.Base.VisitObject(o)
}
