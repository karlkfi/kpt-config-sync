package filesystem

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/transform/tree"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/metadata"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/semantic"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/sync"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/system"
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
	errorBuilder *multierror.Builder,
) []*v1alpha1.Sync {
	root.Accept(tree.NewSystemBuilderVisitor(objects))

	var syncs []*v1alpha1.Sync
	for _, object := range objects {
		switch o := object.Object.(type) {
		case *v1alpha1.Sync:
			syncs = append(syncs, o)
		}
	}

	validateSystem(root, objects, errorBuilder)

	for _, s := range opts.Extension.SyncResources() {
		syncs = append(syncs, s)
		root.System.Objects = append(root.System.Objects, &ast.SystemObject{
			FileObject: ast.FileObject{
				Relative: nomospath.NewFakeRelative("<builtin>"),
				Object:   s,
			},
		})
	}
	return syncs
}

// validateSystem validates objects in system/
func validateSystem(root *ast.Root, objects []ast.FileObject, errorBuilder *multierror.Builder) {
	metadata.DuplicateNameValidatorFactory{}.New(toResourceMetas(objects)).Validate(errorBuilder)

	repoValidator := system.NewRepoVersionValidator()
	root.Accept(repoValidator)
	errorBuilder.Add(repoValidator.Error())

	kindValidator := system.NewKindValidator()
	root.Accept(kindValidator)
	errorBuilder.Add(kindValidator.Error())

	semantic.RepoCountValidator{Objects: objects}.Validate(errorBuilder)

	syncs := fileObjects(objects).syncs()
	sync.KindValidatorFactory.New(syncs).Validate(errorBuilder)
	sync.KnownResourceValidatorFactory(discovery.GetAPIInfo(root)).New(syncs).Validate(errorBuilder)
	sync.NewInheritanceValidatorFactory().New(syncs).Validate(errorBuilder)
	sync.VersionValidatorFactory{}.New(syncs).Validate(errorBuilder)
}
