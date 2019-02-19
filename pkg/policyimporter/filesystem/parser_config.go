package filesystem

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
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
		configs []*v1alpha1.HierarchyConfig,
		vet bool) []ast.Visitor

	// NamespacesDir returns the name of the namespaces dir.
	NamespacesDir() string
}
