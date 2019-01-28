package filesystem

import (
	"os"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/meta"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

// ParserConfig extends the functionality of the parser by allowing the override of visitors or addition
// of sync resources.
// TODO(willbeason): Bespin requires the visitors to be overridden to avoid validators that
// cause a Bespin import to fail, but the resources need to be appended to. This
// isn't great. Ideally the visitors should be able to be chained as well, then
// the ParserOpt could take multiple ParserExtensions and run them all.
type ParserConfig interface {
	// Visitors *overrides* the normal visitor functionality of the parser.
	Visitors(
		syncs []*v1alpha1.Sync,
		clusters []clusterregistry.Cluster,
		selectors []v1alpha1.ClusterSelector,
		vet bool,
		apiInfo *meta.APIInfo) []ast.Visitor

	// ImplicitSyncResources *appends* sync resources to the normal Nomos sync resources.  This is
	// done prior to input type validation, so any type that the user is allowed to specify by default
	// must be returned by this function.
	SyncResources() []*v1alpha1.Sync

	// NamespacesDir returns the name of the namespaces dir.
	NamespacesDir() string
}

// ParserConfigFactory returns the appropriate ParserConfig based on the environment.
func ParserConfigFactory() ParserConfig {
	var e ParserConfig
	// Check for a set environment variable instead of using a flag so as not to expose
	// this WIP externally.
	if _, ok := os.LookupEnv("NOMOS_ENABLE_BESPIN"); ok {
		e = &BespinVisitorProvider{}
	} else {
		e = &NomosVisitorProvider{}
	}
	return e
}
