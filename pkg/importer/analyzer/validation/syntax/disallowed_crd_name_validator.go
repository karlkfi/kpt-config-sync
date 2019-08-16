package syntax

import (
	"fmt"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/util/clusterconfig"
)

// InvalidCRDNameErrorCode is the error code for invalidCRDNameError.
const InvalidCRDNameErrorCode = "1048"

func init() {
	crd := fake.CustomResourceDefinition()
	status.AddExamples(InvalidCRDNameErrorCode, invalidCRDNameError(
		&crd,
	))
}

var invalidCRDNameErrorBuilder = status.NewErrorBuilder(InvalidCRDNameErrorCode)

// invalidCRDNameError reports a CRD with an invalid name in the repo.
func invalidCRDNameError(resource id.Resource) status.Error {
	return invalidCRDNameErrorBuilder.WithResources(resource).Errorf(
		"The CustomResourceDefinition has an invalid name. To fix, change the name to `spec.names.plural+\".\"+spec.group`.")
}

// NewCRDNameValidator returns a validator that validates CRDs have valid names.
func NewCRDNameValidator() *visitor.ValidatorVisitor {
	return visitor.NewClusterObjectValidator(func(o *ast.ClusterObject) status.MultiError {
		return status.From(ValidateCRDName(o.FileObject))
	})
}

// ValidateCRDName returns an error
func ValidateCRDName(o ast.FileObject) status.Error {
	if o.GroupVersionKind() != kinds.CustomResourceDefinition() {
		return nil
	}

	crd, err := clusterconfig.AsCRD(o.Object)
	if err != nil {
		return status.ResourceWrap(err, "could not deserialize CRD", &o)
	}

	expectedName := fmt.Sprintf("%s.%s", crd.Spec.Names.Plural, crd.Spec.Group)
	if crd.Name != expectedName {
		return invalidCRDNameError(&o)
	}

	return nil
}
