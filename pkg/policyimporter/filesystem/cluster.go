package filesystem

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/metadata"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/util/multierror"
)

func validateCluster(objects []ast.FileObject, errorBuilder *multierror.Builder) {
	metadata.Validate(objects, errorBuilder)

	syntax.DisallowSystemObjectsValidator.Validate(objects, errorBuilder)
	syntax.FlatDirectoryValidator.Validate(toSources(objects), errorBuilder)
}

func processCluster(
	objects []ast.FileObject,
	fsRoot *ast.Root) {
	for _, i := range objects {
		fsRoot.Cluster.Objects = append(fsRoot.Cluster.Objects, &ast.ClusterObject{FileObject: i})
	}
}
