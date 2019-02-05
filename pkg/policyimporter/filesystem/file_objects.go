package filesystem

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/sync"
)

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
