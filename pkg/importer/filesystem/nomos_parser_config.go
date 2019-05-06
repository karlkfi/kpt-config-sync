package filesystem

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/transform"
	"github.com/google/nomos/pkg/importer/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/importer/analyzer/validation"
	"github.com/google/nomos/pkg/importer/analyzer/validation/hierarchyconfig"
	"github.com/google/nomos/pkg/importer/analyzer/validation/metadata"
	"github.com/google/nomos/pkg/importer/analyzer/validation/semantic"
	"github.com/google/nomos/pkg/importer/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/importer/analyzer/validation/system"
	"github.com/google/nomos/pkg/importer/analyzer/validation/visitors"
	"github.com/google/nomos/pkg/kinds"
)

var _ ParserConfig = NomosVisitorProvider{}

// NomosVisitorProvider is the default visitor provider.  It handles
// plain vanilla nomos configs.
type NomosVisitorProvider struct {
}

// Visitors implements ParserConfig
func (n NomosVisitorProvider) Visitors(
	configs []*v1.HierarchyConfig,
	vetEnabled bool,
	enableCRDs bool) []ast.Visitor {

	specs := toInheritanceSpecs(configs)
	v := []ast.Visitor{
		&mustSucceed{Visitor: syntax.NewParseValidator()},
		selectors.NewClusterSelectorAdder(),
		system.NewRepoVersionValidator(),
		system.NewKindValidator(),
		system.NewMissingRepoValidator(),
		semantic.NewSingletonResourceValidator(kinds.Repo()),
		hierarchyconfig.NewHierarchyConfigKindValidator(enableCRDs),
		hierarchyconfig.NewKnownResourceValidator(),
		hierarchyconfig.NewInheritanceValidator(),
		visitors.NewSupportedClusterResourcesValidator(enableCRDs),
		syntax.NewClusterRegistryKindValidator(),
		syntax.NewFlatNodeValidator(),
		semantic.NewSingletonResourceValidator(kinds.Namespace()),
		syntax.NewDisallowSystemObjectsValidator(),
		metadata.NewNameValidator(),
		metadata.NewNamespaceAnnotationValidator(),
		metadata.NewNamespaceValidator(),
		syntax.NewDirectoryNameValidator(),
		visitors.NewUniqueDirectoryValidator(),
		syntax.NewNamespaceKindValidator(),
		syntax.NewDisallowedFieldsValidator(),
		metadata.NewAnnotationValidator(),
		metadata.NewManagedAnnotationValidator(),
		metadata.NewLabelValidator(),
		validation.NewInputValidator(specs, vetEnabled),
		semantic.NewAbstractResourceValidator(),
		semantic.NewCRDRemovalValidator(enableCRDs),
		transform.NewPathAnnotationVisitor(),
		validation.NewScope(),
		transform.NewClusterSelectorVisitor(), // Filter out unneeded parts of the tree
		transform.NewAnnotationInlinerVisitor(),
		transform.NewInheritanceVisitor(specs),
		transform.NewEphemeralResourceRemover(),
		metadata.NewDuplicateNameValidator(),
	}
	if spec, found := specs[kinds.ResourceQuota().GroupKind()]; found && spec.Mode == v1.HierarchyModeHierarchicalQuota {
		v = append(v, validation.NewQuotaValidator())
		v = append(v, transform.NewQuotaVisitor())
	}
	return v
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

// NamespacesDir implements ParserConfig
func (n NomosVisitorProvider) NamespacesDir() string {
	return repo.NamespacesDir
}
