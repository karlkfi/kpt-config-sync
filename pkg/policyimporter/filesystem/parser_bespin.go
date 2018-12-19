package filesystem

import (
	bespinv1 "github.com/google/nomos/pkg/api/policyascode/v1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/transform"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation"
	"github.com/google/nomos/pkg/policyimporter/meta"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
)

func init() {
	utilruntime.Must(bespinv1.AddToScheme(legacyscheme.Scheme))
}

var (
	// BespinVisitors is a VisitorProvider that exports the required Bespin visitors.
	BespinVisitors = bespinVisitorProvider{}
	// BespinSyncs are the sync resources required to parse a Bespin repository.
	BespinSyncs = bespinv1.Syncs
)

// bespinVisitorProvider is used when bespin is enabled to handle the bespin specific
// parts that won't pass the regular nomos checks.
type bespinVisitorProvider struct{}

func (b bespinVisitorProvider) visitors(apiInfo *meta.APIInfo) []ast.Visitor {
	// TODO(b/119825336): Bespin and the InputValidator are having trouble playing
	// nicely. For now, just return the visitors that Bespin needs.
	return []ast.Visitor{
		transform.NewGCPHierarchyVisitor(),
		transform.NewGCPPolicyVisitor(),
		validation.NewScope(apiInfo),
		validation.NewNameValidator(),
	}
}
