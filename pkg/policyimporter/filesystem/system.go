package filesystem

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/metadata"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/semantic"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/sync"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	"github.com/google/nomos/pkg/util/discovery"
	"github.com/google/nomos/pkg/util/multierror"
)

// processSystem processes resources in system dir including:
// - Nomos Repo
// - Syncs
// - Reserved Namespaces
func processSystem(
	root *ast.Root,
	objects []ast.FileObject,
	opts ParserOpt,
	errorBuilder *multierror.Builder) (*ast.System, *v1alpha1.Repo, []*v1alpha1.Sync) {
	var syncs []*v1alpha1.Sync
	var repo *v1alpha1.Repo
	sys := &ast.System{}
	for _, object := range objects {
		switch o := object.Object.(type) {
		case *v1alpha1.Repo:
			repo = o
		case *v1alpha1.Sync:
			syncs = append(syncs, o)
		}
		sys.Objects = append(sys.Objects, &ast.SystemObject{FileObject: object})
	}

	validateSystem(root, objects, errorBuilder)

	for _, s := range opts.Extension.SyncResources() {
		syncs = append(syncs, s)
		sys.Objects = append(sys.Objects, &ast.SystemObject{
			FileObject: ast.FileObject{
				Relative: nomospath.NewFakeRelative("<builtin>"),
				Object:   s,
			},
		})
	}
	return sys, repo, syncs
}

// validateSystem validates objects in system/
func validateSystem(root *ast.Root, objects []ast.FileObject, errorBuilder *multierror.Builder) {
	metadata.DuplicateNameValidatorFactory{}.New(toResourceMetas(objects)).Validate(errorBuilder)
	syntax.RepoVersionValidator.Validate(objects, errorBuilder)
	syntax.SystemKindValidator.Validate(objects, errorBuilder)

	semantic.RepoCountValidator{Objects: objects}.Validate(errorBuilder)

	syncs := fileObjects(objects).syncs()
	sync.KindValidatorFactory.New(syncs).Validate(errorBuilder)
	sync.KnownResourceValidatorFactory(discovery.GetAPIInfo(root)).New(syncs).Validate(errorBuilder)
	sync.NewInheritanceValidatorFactory().New(syncs).Validate(errorBuilder)
	sync.VersionValidatorFactory{}.New(syncs).Validate(errorBuilder)
}
