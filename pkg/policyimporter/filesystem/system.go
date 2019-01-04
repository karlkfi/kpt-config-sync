package filesystem

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/metadata"
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

	if opts.Extension != nil {
		syncs = append(syncs, opts.Extension.SyncResources()...)
	}
	return repo, syncs, reservedNamespaces
}

// validateSystem validates objects in system/
func validateSystem(objects []ast.FileObject, repo *v1alpha1.Repo, apiInfo *meta.APIInfo, errorBuilder *multierror.Builder) {
	metadata.Validate(toResourceMetas(objects), errorBuilder)
	syntax.FlatDirectoryValidator.Validate(toSources(objects), errorBuilder)
	syntax.RepoVersionValidator.Validate(objects, errorBuilder)
	syntax.SystemKindValidator.Validate(objects, errorBuilder)

	semantic.RepoCountValidator{Objects: objects}.Validate(errorBuilder)
	semantic.ConfigMapCountValidator{Objects: objects}.Validate(errorBuilder)

	syncs := fileObjects(objects).syncs()
	sync.KindValidatorFactory.New(syncs).Validate(errorBuilder)
	sync.KnownResourceValidatorFactory(apiInfo).New(syncs).Validate(errorBuilder)
	sync.NewInheritanceValidatorFactory(repo).New(syncs).Validate(errorBuilder)
	sync.VersionValidatorFactory{}.New(syncs).Validate(errorBuilder)
}
