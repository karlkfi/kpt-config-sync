package filesystem

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/transform"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/hierarchyconfig"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/metadata"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/semantic"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/system"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/visitors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NomosVisitorProvider is the default visitor provider.  It handles
// plain vanilla nomos configs.
type NomosVisitorProvider struct {
}

// Visitors implements ParserConfig
func (n NomosVisitorProvider) Visitors(
	configs []*v1alpha1.HierarchyConfig,
	syncs []*v1alpha1.Sync,
	vetEnabled bool) []ast.Visitor {

	var specs map[schema.GroupKind]*transform.InheritanceSpec
	if len(configs) > 0 {
		specs = toInheritanceSpecs(configs)
	} else {
		specs = syncsToInheritanceSpecs(syncs)
	}
	v := []ast.Visitor{
		system.NewRepoVersionValidator(),
		system.NewKindValidator(),
		system.NewMissingRepoValidator(),
		semantic.NewSingletonResourceValidator(kinds.Repo()),
		hierarchyconfig.NewHierarchyConfigKindValidator(),
		hierarchyconfig.NewKnownResourceValidator(),
		hierarchyconfig.NewInheritanceValidator(),
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
		validation.NewInputValidator(syncs, specs, vetEnabled),
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
	v = append(v,
		// TODO: remove NewSyncRemover once syncs are disallowed in system/
		transform.NewSyncRemover(),
		transform.NewSyncGenerator(),
	)
	return v
}

// SyncResources implements ParserConfig
func (n NomosVisitorProvider) SyncResources() []*v1alpha1.Sync {
	return nil
}

// NamespacesDir implements ParserConfig
func (n NomosVisitorProvider) NamespacesDir() string {
	return repo.NamespacesDir
}
