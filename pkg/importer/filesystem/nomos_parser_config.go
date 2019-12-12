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

// Visitors implements ParserConfig
func Visitors(configs []*v1.HierarchyConfig) []ast.Visitor {

	specs := toInheritanceSpecs(configs)
	return []ast.Visitor{
		&mustSucceed{Visitor: syntax.NewParseValidator()},
		system.NewRepoVersionValidator(),
		system.NewKindValidator(),
		system.NewMissingRepoValidator(),
		semantic.NewSingletonResourceValidator(kinds.Repo()),
		hierarchyconfig.NewHierarchyConfigKindValidator(),
		hierarchyconfig.NewInheritanceValidator(),
		visitors.NewSupportedClusterResourcesValidator(),
		syntax.NewClusterRegistryKindValidator(),
		semantic.NewSingletonResourceValidator(kinds.Namespace()),
		syntax.NewDisallowSystemObjectsValidator(),
		syntax.NewDeprecatedGroupKindValidator(),
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

// mustSucceed wraps a Visitor, allowing NomosVisitorProvider to specify whether a visitor should
// be fatal if it returns errors.
type mustSucceed struct {
	ast.Visitor
}

var _ ast.Visitor = &mustSucceed{}

// Fatal returns true if the Visitor encountered any errors.
func (m mustSucceed) Fatal() bool {
	return m.Error() != nil
}
