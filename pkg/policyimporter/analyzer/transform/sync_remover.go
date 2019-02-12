package transform

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
)

// SyncRemover removes all sync objects that have been declared.
type SyncRemover struct {
	// Copying is used for copying parts of the ast.Root tree and continuing underlying visitor iteration.
	*visitor.Copying
}

// NewSyncRemover returns a new SyncRemover transform.
func NewSyncRemover() *SyncRemover {
	v := &SyncRemover{
		Copying: visitor.NewCopying(),
	}
	v.Copying.SetImpl(v)
	return v
}

// VisitSystemObject implements Visitor
func (v *SyncRemover) VisitSystemObject(o *ast.SystemObject) *ast.SystemObject {
	if _, ok := o.FileObject.Object.(*v1alpha1.Sync); ok {
		return nil
	}
	return o
}
