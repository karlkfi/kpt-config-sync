package filesystem

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/metadata"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/util/multierror"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

func validateClusterRegistry(objects []ast.FileObject, errorBuilder *multierror.Builder) {
	metadata.DuplicateNameValidatorFactory{}.New(toResourceMetas(objects)).Validate(errorBuilder)
	syntax.ClusterregistryKindValidator.Validate(objects, errorBuilder)
	syntax.FlatDirectoryValidator.Validate(ast.ToRelative(objects), errorBuilder)
}

func getClusters(objects []ast.FileObject) []clusterregistry.Cluster {
	var clusters []clusterregistry.Cluster
	for _, object := range objects {
		switch o := object.Object.(type) {
		case *clusterregistry.Cluster:
			clusters = append(clusters, *o)
		}
	}
	return clusters
}

// processClusterRegistryDir looks at all files in <root>/clusterregistry and
// extracts Cluster and ClusterSelector objects out.
func getSelectors(objects []ast.FileObject) []v1alpha1.ClusterSelector {
	var selectors []v1alpha1.ClusterSelector
	for _, object := range objects {
		switch o := object.Object.(type) {
		case *v1alpha1.ClusterSelector:
			selectors = append(selectors, *o)
		}
	}
	return selectors
}

func getClusterRegistry(objects []ast.FileObject) *ast.ClusterRegistry {
	cr := &ast.ClusterRegistry{}
	for _, o := range objects {
		cr.Objects = append(cr.Objects, &ast.ClusterRegistryObject{FileObject: o})
	}
	return cr
}
