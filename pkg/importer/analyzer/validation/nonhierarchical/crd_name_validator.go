package nonhierarchical

import (
	"fmt"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/clusterconfig"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// InvalidCRDNameErrorCode is the error code for InvalidCRDNameError.
const InvalidCRDNameErrorCode = "1048"

var invalidCRDNameErrorBuilder = status.NewErrorBuilder(InvalidCRDNameErrorCode)

// InvalidCRDNameError reports a CRD with an invalid name in the repo.
func InvalidCRDNameError(resource client.Object, expected string) status.Error {
	return invalidCRDNameErrorBuilder.
		Sprintf("The CustomResourceDefinition `metadata.name` MUST be in the form: `<spec.names.plural>.<spec.group>`. "+
			"To fix, update those fields or change `metadata.name` to %q.",
			expected).
		BuildWithResources(resource)
}

// ValidateCRDName returns an error
func ValidateCRDName(o ast.FileObject) status.Error {
	if o.GetObjectKind().GroupVersionKind().GroupKind() != kinds.CustomResourceDefinition() {
		return nil
	}

	crd, err := clusterconfig.AsCRD(o.Object)
	if err != nil {
		return err
	}
	expectedName := fmt.Sprintf("%s.%s", crd.Spec.Names.Plural, crd.Spec.Group)
	if crd.Name != expectedName {
		return InvalidCRDNameError(&o, expectedName)
	}

	return nil
}
