package nonhierarchical

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/semantic"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/clusterconfig"
)

// CRDRemovalValidator ensures the repo doesn't declare resources which use now-nonexistent CRDs.
func CRDRemovalValidator(crdInfo *clusterconfig.CRDInfo) Validator {
	return perObjectValidator(func(o ast.FileObject) status.Error {
		return semantic.CheckCRDPendingRemoval(crdInfo, o)
	})
}
