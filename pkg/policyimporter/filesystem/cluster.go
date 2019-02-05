package filesystem

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/metadata"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/util/multierror"
)

func validateCluster(objects []ast.FileObject, errorBuilder *multierror.Builder) {
	metadata.DuplicateNameValidatorFactory{}.New(toResourceMetas(objects)).Validate(errorBuilder)
	syntax.FlatDirectoryValidator.Validate(ast.ToRelative(objects), errorBuilder)
}

func processCluster(
	objects []ast.FileObject,
	fsRoot *ast.Root) {
	for _, i := range objects {
		fsRoot.Cluster.Objects = append(fsRoot.Cluster.Objects, &ast.ClusterObject{FileObject: i})
	}
}
