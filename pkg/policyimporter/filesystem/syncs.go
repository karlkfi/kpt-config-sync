package filesystem

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/sync"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	"github.com/google/nomos/pkg/util/discovery"
	"github.com/google/nomos/pkg/util/multierror"
)

// processSyncs processes Sync resources.
// TODO: Delete once syncs are defunct.
func processSyncs(
	root *ast.Root,
	objects []ast.FileObject,
	opts ParserOpt,
) []*v1alpha1.Sync {
	var syncs []*v1alpha1.Sync
	for _, object := range objects {
		switch o := object.Object.(type) {
		case *v1alpha1.Sync:
			syncs = append(syncs, o)
		}
	}

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

// validateSyncs validates Syncs.
// TODO: Delete once syncs are defunct.
func validateSyncs(root *ast.Root, objects []ast.FileObject, eb *multierror.Builder) {
	syncs := syncsFrom(objects)
	sync.KindValidatorFactory.New(syncs).Validate(eb)
	sync.KnownResourceValidatorFactory(discovery.GetAPIInfo(root)).New(syncs).Validate(eb)
	sync.NewInheritanceValidatorFactory().New(syncs).Validate(eb)
	sync.VersionValidatorFactory{}.New(syncs).Validate(eb)
}

// syncs returns all Syncs contained in a fileObjects as []sync.FileSync.
// TODO: Delete once syncs are defunct.
func syncsFrom(objects []ast.FileObject) []sync.FileSync {
	var result []sync.FileSync

	for _, obj := range objects {
		switch o := obj.Object.(type) {
		case *v1alpha1.Sync:
			result = append(result, sync.NewFileSync(o, obj.Relative))
		}
	}

	return result
}
