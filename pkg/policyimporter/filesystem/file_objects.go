package filesystem

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/semantic"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/sync"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/util/multierror"
)

// validation run on all objects
func validateObjects(objects []ast.FileObject, errorBuilder *multierror.Builder) {
	syntax.AnnotationValidator.Validate(objects, errorBuilder)
	syntax.LabelValidator.Validate(objects, errorBuilder)
	syntax.MetadataNamespaceValidator.Validate(objects, errorBuilder)
	syntax.MetadataNameValidator.Validate(objects, errorBuilder)

	semantic.DuplicateNameValidator{Objects: objects}.Validate(errorBuilder)
}

// toSources extracts the sources from the set of provided FileObjects.
// TODO(willbeason): Move to fileObjects
func toSources(infos []ast.FileObject) []string {
	result := make([]string, len(infos))
	for i, info := range infos {
		result[i] = info.Source
	}
	return result
}

// fileObjects extends []ast.FileObject to provide operations helpful for parsing.
type fileObjects []ast.FileObject

// syncs returns all Syncs contained in a fileObjects as []sync.FileSync.
func (objects fileObjects) syncs() []sync.FileSync {
	var result []sync.FileSync

	for _, obj := range objects {
		switch o := obj.Object.(type) {
		case *v1alpha1.Sync:
			result = append(result, sync.FileSync{Source: obj.Source, Sync: o})
		}
	}

	return result
}
