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
	"fmt"
	"path"
	"strings"

	"github.com/google/nomos/pkg/api/policyhierarchy"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	sels "github.com/google/nomos/pkg/policyimporter/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/policyimporter/reserved"
	"github.com/google/nomos/pkg/util/multierror"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

// ClusterCoverage contains information about which clusters are covered by which cluster
// selectors.
type ClusterCoverage struct {
	clusterNames     map[string]bool
	coveredClusters  map[string]bool
	selectorNames    map[string]bool
	coveredSelectors map[string]bool
}

func newClusterCoverage(
	clusters []clusterregistry.Cluster,
	selectors []v1alpha1.ClusterSelector,
	errs *multierror.Builder,
) *ClusterCoverage {
	cov := ClusterCoverage{
		clusterNames:     map[string]bool{},
		coveredClusters:  map[string]bool{},
		selectorNames:    map[string]bool{},
		coveredSelectors: map[string]bool{},
	}
	for _, c := range clusters {
		cov.clusterNames[c.ObjectMeta.Name] = true
	}
	for _, s := range selectors {
		cov.selectorNames[s.ObjectMeta.Name] = true
	}
	for _, s := range selectors {
		sn := s.ObjectMeta.Name
		selector, err := sels.AsPopulatedSelector(&s.Spec.Selector)
		if err != nil {
			errs.Add(InvalidSelector{sn, err})
			continue
		}
		for _, c := range clusters {
			cn := c.ObjectMeta.Name
			if sels.IsSelected(c.ObjectMeta.Labels, selector) {
				cov.coveredClusters[cn] = true
				cov.coveredSelectors[sn] = true
			}
		}
	}
	return &cov
}

// ValidateObject validates the coverage of the object with clusters and selectors. An object
// may not have an annotation, but if it does, it has to map to a valid selector.  Also if an
// object has a selector in the annotation, that annotation must refer to a valid selector.
func (c ClusterCoverage) ValidateObject(o metav1.Object, errs *multierror.Builder) {
	a := v1alpha1.GetClusterSelectorAnnotation(o.GetAnnotations())
	if a == "" {
		return
	}
	if !c.selectorNames[a] {
		errs.Add(ObjectHasUnknownClusterSelector{o, a})
	}
}

// InputValidator checks various filesystem constraints after loading into the tree format.
// Error messages emitted from the validator should be formatted to first print the constraint
// that is being violated, then print a useful error message on what is violating the constraint
// and what is required to fix it.
type InputValidator struct {
	*visitor.Base
	errs               multierror.Builder
	reserved           *reserved.Namespaces
	dirNames           map[string]*ast.TreeNode
	nodes              []*ast.TreeNode
	seenResourceQuotas map[string]struct{}
	allowedGVKs        map[schema.GroupVersionKind]struct{}
	coverage           *ClusterCoverage
}

// InputValidator implements ast.Visitor
var _ ast.Visitor = &InputValidator{}

// NewInputValidator creates a new validator.  allowedGVKs represents the set
// of valid group-version-kinds for objects in the namespaces and cluster
// directories.  Objects of other types will be treated as an error. clusters
// is the list of clusters defined in the source of truth, and cs is the list
// of selectors.  vet turns on "vetting mode", a mode of stricter control for use
// in nomosvet.
func NewInputValidator(
	allowedGVKs map[schema.GroupVersionKind]struct{},
	clusters []clusterregistry.Cluster,
	cs []v1alpha1.ClusterSelector,
	vet bool) *InputValidator {
	v := &InputValidator{
		Base:               visitor.NewBase(),
		reserved:           reserved.EmptyNamespaces(),
		dirNames:           make(map[string]*ast.TreeNode),
		seenResourceQuotas: make(map[string]struct{}),
		allowedGVKs:        allowedGVKs,
	}
	v.Base.SetImpl(v)

	if vet {
		v.coverage = newClusterCoverage(clusters, cs, &v.errs)
		//		v.coverage.ValidateCoverage(&v.errs)
	}
	return v
}

// Error returns any errors encountered during processing
func (v *InputValidator) Error() error {
	return v.errs.Build()
}

// VisitReservedNamespaces implements Visitor
func (v *InputValidator) VisitReservedNamespaces(rs *ast.ReservedNamespaces) ast.Node {
	if r, err := reserved.From(&rs.ConfigMap); err != nil {
		v.errs.Add(err)
	} else {
		v.reserved = r
	}
	return nil
}

// VisitTreeNode implements Visitor
func (v *InputValidator) VisitTreeNode(n *ast.TreeNode) ast.Node {
	name := path.Base(n.Path)
	if v.reserved.IsReserved(name) {
		// The namespace's name must not be a reserved namespace name.
		v.errs.Add(ReservedDirectoryNameError{n})
	}
	if other, found := v.dirNames[name]; found {
		// The namespace must not duplicate the name of another namespace.
		v.errs.Add(DuplicateDirectoryNameError{this: n, other: other})
	} else {
		v.dirNames[name] = n
	}
	if len(v.nodes) != 0 {
		// Namespaces may not have children.
		if parent := v.nodes[len(v.nodes)-1]; parent.Type == ast.Namespace {
			v.errs.Add(IllegalNamespaceSubdirectoryError{child: n, parent: parent})
		}
	}
	if n.Type == ast.Namespace {
		if _, found := n.Annotations[v1alpha1.NamespaceSelectorAnnotationKey]; found {
			// Namespaces may not use the selector annotation.
			v.errs.Add(IllegalNamespaceSelectorAnnotationError{n})
		}
	}
	for _, s := range n.Selectors {
		if err := v.checkNamespaceSelectorAnnotations(s); err != nil {
			v.errs.Add(err)
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
func (v *InputValidator) checkNamespaceSelectorAnnotations(s *v1alpha1.NamespaceSelector) error {
	a := s.GetAnnotations()
	if a == nil {
		return nil
	}
	if _, ok := a[v1alpha1.ClusterSelectorAnnotationKey]; ok {
		return NamespaceSelectorMayNotHaveAnnotation{s}
	}
	return nil
}

func (v *InputValidator) checkAnnotationsAndLabels(o ast.FileObject) {
	if err := v.checkAnnotations(o); err != nil {
		v.errs.Add(err)
	}
	if err := v.checkLabels(o); err != nil {
		v.errs.Add(err)
	}
}

// VisitClusterObject implements Visitor
func (v *InputValidator) VisitClusterObject(o *ast.ClusterObject) ast.Node {
	gvk := o.GroupVersionKind()
	if _, found := v.allowedGVKs[gvk]; !found {
		v.errs.Add(UnsyncableClusterObjectError{o})
	}
	// The cluster/ directory may not have children.
	if dir := path.Dir(o.Source); dir != repo.ClusterDir {
		// As implemented, only prints a message if an offending directory defines an object.
		v.errs.Add(IllegalClusterSubdirectoryError{dir})
	}
	v.checkAnnotationsAndLabels(o.FileObject)
	if v.coverage != nil {
		v.coverage.ValidateObject(o.ToMeta(), &v.errs)
	}
	return v.Base.VisitClusterObject(o)
}

// VisitObject implements Visitor
func (v *InputValidator) VisitObject(o *ast.NamespaceObject) ast.Node {

	// checkSingleResourceQuota ensures that at most one ResourceQuota resource is present in each
	// directory.
	if o.GroupVersionKind() == corev1.SchemeGroupVersion.WithKind("ResourceQuota") {
		curPath := v.nodes[len(v.nodes)-1].Path
		if _, found := v.seenResourceQuotas[curPath]; found {
			v.errs.Add(ConflictingResourceQuotaError{o})
		} else {
			v.seenResourceQuotas[curPath] = struct{}{}
		}
	}

	if _, found := v.allowedGVKs[o.GroupVersionKind()]; !found {
		v.errs.Add(UnsyncableNamespaceObjectError{o})
	}

	node := v.nodes[len(v.nodes)-1]
	if node.Type == ast.AbstractNamespace {
		switch o.GroupVersionKind() {
		case rbacv1.SchemeGroupVersion.WithKind("RoleBinding"):
		case corev1.SchemeGroupVersion.WithKind("ResourceQuota"):
		default:
			v.errs.Add(IllegalAbstractNamespaceObjectKindError{o})
		}
	}

	v.checkAnnotationsAndLabels(o.FileObject)
	if v.coverage != nil {
		v.coverage.ValidateObject(o.ToMeta(), &v.errs)
	}

	return v.Base.VisitObject(o)
}

func invalids(m map[string]string, allowed map[string]struct{}) []string {
	var found []string

	for k := range m {
		if _, found := allowed[k]; found {
			continue
		}
		if strings.HasPrefix(k, policyhierarchy.GroupName+"/") {
			found = append(found, fmt.Sprintf("%q", k))
		}
	}

	return found
}

func (v *InputValidator) checkAnnotations(o ast.FileObject) error {
	found := invalids(o.ToMeta().GetAnnotations(), v1alpha1.InputAnnotations)
	if len(found) == 0 {
		return nil
	}
	return IllegalAnnotationDefinitionError{o, found}
}

var noneAllowed = map[string]struct{}{}

func (v *InputValidator) checkLabels(o ast.FileObject) error {
	found := invalids(o.ToMeta().GetLabels(), noneAllowed)
	if len(found) == 0 {
		return nil
	}
	return IllegalLabelDefinitionError{o, found}
}
