package nonhierarchical

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
)

// KptfileExistenceValidator checks for the existence of Kptfile(s).
var KptfileExistenceValidator = PerObjectValidator(kptfileExistence)

func kptfileExistence(o ast.FileObject) status.Error {
	if o.GroupVersionKind().GroupKind() == kinds.KptFile().GroupKind() {
		return KptfileExistenceError(&o)
	}
	return nil
}

// KptfileExistenceErrorCode is the error code for KptfileExistenceError.
const KptfileExistenceErrorCode = "1063"

var kptfileExistenceError = status.NewErrorBuilder(KptfileExistenceErrorCode)

// KptfileExistenceError reports the existence of Kptfile.
func KptfileExistenceError(resource id.Resource) status.Error {
	return kptfileExistenceError.
		Sprintf("Found Kptfile(s) in the Root Repo. Kptfile(s) are only supported in Namespace Repos. To fix, remove the Kptfile(s) from the Root Repo.").
		BuildWithResources(resource)
}
