package filesystem

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/transform"
	"github.com/google/nomos/pkg/importer/analyzer/validation"
	"github.com/google/nomos/pkg/importer/analyzer/validation/hierarchyconfig"
	"github.com/google/nomos/pkg/importer/analyzer/validation/metadata"
	"github.com/google/nomos/pkg/importer/analyzer/validation/semantic"
	"github.com/google/nomos/pkg/importer/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/importer/analyzer/validation/system"
	"github.com/google/nomos/pkg/importer/analyzer/validation/visitors"
	"github.com/google/nomos/pkg/kinds"
)

// hierarchicalVisitors returns the set of Visitors used specifically for
// validating hierarchical repositories.
func hierarchicalVisitors(configs []*v1.HierarchyConfig) []ast.Visitor {

	specs := toInheritanceSpecs(configs)
	return []ast.Visitor{
		system.NewRepoVersionValidator(),
		system.NewMissingRepoValidator(),
		semantic.NewSingletonResourceValidator(kinds.Repo()),
		hierarchyconfig.NewHierarchyConfigKindValidator(),
		hierarchyconfig.NewInheritanceValidator(),
		visitors.NewSupportedClusterResourcesValidator(),
		semantic.NewSingletonResourceValidator(kinds.Namespace()),
		syntax.NewDisallowSystemObjectsValidator(),
		metadata.NewNamespaceDirectoryNameValidator(),
		metadata.NewMetadataNamespaceDirectoryValidator(),
		syntax.NewDirectoryNameValidator(),
		syntax.NewNamespaceKindValidator(),
		metadata.NewAnnotationValidator(),
		metadata.NewLabelValidator(),
		validation.NewInputValidator(specs),
		semantic.NewAbstractResourceValidator(),
		transform.NewPathAnnotationVisitor(),
		transform.NewInheritanceVisitor(),
	}
}
