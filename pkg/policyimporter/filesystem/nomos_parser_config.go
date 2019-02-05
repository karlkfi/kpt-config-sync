package filesystem

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/transform"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/policyimporter/meta"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

// NomosVisitorProvider is the default visitor provider.  It handles
// plain vanilla nomos configs.
type NomosVisitorProvider struct {
}

// Visitors implements ParserConfig
func (n NomosVisitorProvider) Visitors(
	syncs []*v1alpha1.Sync,
	clusters []clusterregistry.Cluster,
	selectors []v1alpha1.ClusterSelector,
	vet bool,
	apiInfo *meta.APIInfo) []ast.Visitor {
	specs := toInheritanceSpecs(syncs)
	visitors := []ast.Visitor{
		syntax.NewNamespaceKindValidator(),
		validation.NewInputValidator(syncs, specs, clusters, selectors, vet),
		transform.NewPathAnnotationVisitor(),
		validation.NewScope(apiInfo),
		transform.NewClusterSelectorVisitor(), // Filter out unneeded parts of the tree
		transform.NewAnnotationInlinerVisitor(),
		transform.NewInheritanceVisitor(specs),
		transform.NewEphemeralResourceRemover(),
	}
	if spec, found := specs[kinds.ResourceQuota().GroupKind()]; found && spec.Mode == v1alpha1.HierarchyModeHierarchicalQuota {
		visitors = append(visitors, validation.NewQuotaValidator())
		visitors = append(visitors, transform.NewQuotaVisitor())
	}
	visitors = append(visitors, validation.NewNameValidator())

	return visitors
}

// SyncResources implements ParserConfig
func (n NomosVisitorProvider) SyncResources() []*v1alpha1.Sync {
	return nil
}

// NamespacesDir implements ParserConfig
func (n NomosVisitorProvider) NamespacesDir() string {
	return repo.NamespacesDir
}
