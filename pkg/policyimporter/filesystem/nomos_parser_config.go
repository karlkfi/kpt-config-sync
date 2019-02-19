package filesystem

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/transform"
	"github.com/google/nomos/pkg/policyimporter/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/hierarchyconfig"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/metadata"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/semantic"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/system"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/visitors"
)

// NomosVisitorProvider is the default visitor provider.  It handles
// plain vanilla nomos configs.
type NomosVisitorProvider struct {
}

// Visitors implements ParserConfig
func (n NomosVisitorProvider) Visitors(
	configs []*v1alpha1.HierarchyConfig,
	vetEnabled bool) []ast.Visitor {

	specs := toInheritanceSpecs(configs)
	v := []ast.Visitor{
		selectors.NewClusterSelectorAdder(),
		system.NewRepoVersionValidator(),
		system.NewKindValidator(),
		system.NewMissingRepoValidator(),
		semantic.NewSingletonResourceValidator(kinds.Repo()),
		hierarchyconfig.NewHierarchyConfigKindValidator(),
		hierarchyconfig.NewKnownResourceValidator(),
		hierarchyconfig.NewInheritanceValidator(),
		visitors.NewSupportedClusterResourcesValidator(),
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
		metadata.NewAnnotationValidator(),
		metadata.NewLabelValidator(),
		validation.NewInputValidator(specs, vetEnabled),
		transform.NewPathAnnotationVisitor(),
		validation.NewScope(),
		transform.NewClusterSelectorVisitor(), // Filter out unneeded parts of the tree
		transform.NewAnnotationInlinerVisitor(),
		transform.NewInheritanceVisitor(specs),
		transform.NewEphemeralResourceRemover(),
		metadata.NewDuplicateNameValidator(),
	}
	if spec, found := specs[kinds.ResourceQuota().GroupKind()]; found && spec.Mode == v1alpha1.HierarchyModeHierarchicalQuota {
		v = append(v, validation.NewQuotaValidator())
		v = append(v, transform.NewQuotaVisitor())
	}
	return v
}

// NamespacesDir implements ParserConfig
func (n NomosVisitorProvider) NamespacesDir() string {
	return repo.NamespacesDir
}
