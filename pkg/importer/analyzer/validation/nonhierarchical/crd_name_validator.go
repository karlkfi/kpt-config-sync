package nonhierarchical

import (
	"fmt"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/clusterconfig"
)

// CRDNameValidator validates that CRDs have the expected metadata.name.
var CRDNameValidator = PerObjectValidator(validateCRDName)

// InvalidCRDNameErrorCode is the error code for InvalidCRDNameError.
const InvalidCRDNameErrorCode = "1048"

var invalidCRDNameErrorBuilder = status.NewErrorBuilder(InvalidCRDNameErrorCode)

// InvalidCRDNameError reports a CRD with an invalid name in the repo.
func InvalidCRDNameError(resource id.Resource) status.Error {
	return invalidCRDNameErrorBuilder.
		Sprintf("The CustomResourceDefinition has an invalid name. To fix, change the name to `spec.names.plural+\".\"+spec.group`.").
		BuildWithResources(resource)
}

// validateCRDName returns an error
func validateCRDName(o ast.FileObject) status.Error {
	if o.GroupVersionKind().GroupKind() != kinds.CustomResourceDefinition() {
		return nil
	}

	crd, err := clusterconfig.AsCRD(o.Object)
	if err != nil {
		return status.ResourceWrap(err, "could not deserialize CRD", &o)
	}

	expectedName := fmt.Sprintf("%s.%s", crd.Spec.Names.Plural, crd.Spec.Group)
	if crd.Name != expectedName {
		return InvalidCRDNameError(&o)
	}

	return nil
}
