package transform

import (
	"sort"

	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"

	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// SyncGenerator transforms Sync objects so they only contain one group version.
type SyncGenerator struct {
	// Copying is used for copying parts of the ast.Root tree and continuing underlying visitor iteration.
	*visitor.Copying

	allGk map[schema.GroupKind]struct{}
}

// NewSyncGenerator returns a new SyncGenerator transform.
func NewSyncGenerator() *SyncGenerator {
	v := &SyncGenerator{
		Copying: visitor.NewCopying(),
		allGk:   map[schema.GroupKind]struct{}{},
	}
	v.Copying.SetImpl(v)
	return v
}

// VisitRoot implements Visitor
func (v *SyncGenerator) VisitRoot(r *ast.Root) *ast.Root {
	nr := v.Copying.VisitRoot(r)
	var gkList []schema.GroupKind
	for gk := range v.allGk {
		gkList = append(gkList, gk)
	}
	sort.Slice(gkList, func(i, j int) bool {
		return gkList[i].String() < gkList[j].String()
	})
	for _, gk := range gkList {
		nr.SystemObjects = append(nr.SystemObjects, &ast.SystemObject{
			FileObject: ast.FileObject{
				Object: v.genSync(gk),
			},
		})
	}
	return nr
}

// VisitClusterObject implements Visitor
func (v *SyncGenerator) VisitClusterObject(o *ast.ClusterObject) *ast.ClusterObject {
	v.allGk[o.GroupVersionKind().GroupKind()] = struct{}{}
	return o
}

// VisitObject implements Visitor
func (v *SyncGenerator) VisitObject(o *ast.NamespaceObject) *ast.NamespaceObject {
	v.allGk[o.GroupVersionKind().GroupKind()] = struct{}{}
	return o
}

func (v *SyncGenerator) genSync(gk schema.GroupKind) *v1.Sync {
	return v1.NewSync(gk.Group, gk.Kind)
}
