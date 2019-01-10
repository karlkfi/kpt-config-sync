package filesystem

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/metadata"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/sync"
)

// toSources extracts the sources from the set of provided FileObjects.
// TODO(willbeason): Move to fileObjects
func toSources(infos []ast.FileObject) []string {
	result := make([]string, len(infos))
	for i, info := range infos {
		result[i] = info.RelativeSlashPath()
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
			result = append(result, sync.NewFileSync(o, obj.Relative))
		}
	}

	return result
}

func toResourceMetas(objects []ast.FileObject) []metadata.ResourceMeta {
	metas := make([]metadata.ResourceMeta, len(objects))
	for i := range objects {
		metas[i] = &objects[i]
	}
	return metas
}
