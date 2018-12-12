package filesystem

import (
	bespinv1 "github.com/google/nomos/pkg/api/policyascode/v1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/semantic"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/sync"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/policyimporter/meta"
	"github.com/google/nomos/pkg/util/multierror"
	corev1 "k8s.io/api/core/v1"
)

// processSystem processes resources in system dir including:
// - Nomos Repo
// - Syncs
// - Reserved Namespaces
func processSystem(
	objects []ast.FileObject,
	opts ParserOpt,
	apiInfo *meta.APIInfo, errorBuilder *multierror.Builder) (*v1alpha1.Repo, []*v1alpha1.Sync, *ast.ReservedNamespaces) {
	var syncs []*v1alpha1.Sync
	var repo *v1alpha1.Repo
	var reservedNamespaces *ast.ReservedNamespaces
	for _, object := range objects {
		switch o := object.Object.(type) {
		case *v1alpha1.Repo:
			repo = o
		case *corev1.ConfigMap:
			reservedNamespaces = &ast.ReservedNamespaces{ConfigMap: *o}
		case *v1alpha1.Sync:
			syncs = append(syncs, o)
		}
	}

	validateSystem(objects, repo, apiInfo, errorBuilder)

	if opts.Bespin {
		syncs = append(syncs, bespinv1.Syncs...)
	}
	return repo, syncs, reservedNamespaces
}

// validateSystem validates objects in system/
func validateSystem(objects []ast.FileObject, repo *v1alpha1.Repo, apiInfo *meta.APIInfo, errorBuilder *multierror.Builder) {
	validateObjects(objects, errorBuilder)
	syntax.FlatDirectoryValidator.Validate(toSources(objects), errorBuilder)
	syntax.RepoVersionValidator.Validate(objects, errorBuilder)
	syntax.SystemKindValidator.Validate(objects, errorBuilder)

	semantic.RepoCountValidator{Objects: objects}.Validate(errorBuilder)
	semantic.ConfigMapCountValidator{Objects: objects}.Validate(errorBuilder)

	sync.KindValidator.Validate(objects, errorBuilder)
	sync.KnownResourceValidator{APIInfo: apiInfo}.Validate(objects, errorBuilder)
	sync.InheritanceValidator{Repo: repo}.Validate(objects, errorBuilder)
	sync.VersionValidator{Objects: objects}.Validate(errorBuilder)
}
