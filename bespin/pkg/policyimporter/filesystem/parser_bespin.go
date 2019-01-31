package filesystem

import (
	bespinv1 "github.com/google/nomos/bespin/pkg/api/bespin/v1"
	"github.com/google/nomos/bespin/pkg/policyimporter/analyzer/transform"
	bespinvalidation "github.com/google/nomos/bespin/pkg/validation"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation"
	"github.com/google/nomos/pkg/policyimporter/meta"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
)

func init() {
	utilruntime.Must(bespinv1.AddToScheme(legacyscheme.Scheme))
}

// BespinVisitorProvider is used when bespin is enabled to handle the bespin specific
// parts that won't pass the regular nomos checks.
type BespinVisitorProvider struct{}

// Visitors implements ParserConfig
func (b BespinVisitorProvider) Visitors(
	syncs []*v1alpha1.Sync,
	clusters []clusterregistry.Cluster,
	selectors []v1alpha1.ClusterSelector,
	vet bool,
	apiInfo *meta.APIInfo) []ast.Visitor {
	// TODO(b/119825336): Bespin and the InputValidator are having trouble playing
	// nicely. For now, just return the visitors that Bespin needs.
	return []ast.Visitor{
		transform.NewGCPHierarchyVisitor(),
		transform.NewGCPPolicyVisitor(),
		transform.NewGCPAnnotationVisitor(),
		validation.NewScope(apiInfo),
		validation.NewNameValidator(),
		bespinvalidation.NewUniqueIAMValidator(),
		bespinvalidation.NewMaxFolderDepthValidator(),
		bespinvalidation.NewIAMFilenameValidator(),
	}
}

// SyncResources implements ParserConfig
func (b BespinVisitorProvider) SyncResources() []*v1alpha1.Sync {
	return bespinv1.Syncs
}

// NamespacesDir implements ParserConfig
func (b BespinVisitorProvider) NamespacesDir() string {
	return "hierarchy"
}
