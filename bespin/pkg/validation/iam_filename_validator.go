package validation

import (
	"github.com/google/nomos/bespin/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
)

const allowedIAMFilename = "gcp-iam-policy.yaml"

// NewIAMFilenameValidator creates a new validator that ensures IAMPolicies are only declared in files
// with a particular filename.
func NewIAMFilenameValidator() *visitor.ValidatorVisitor {
	return visitor.NewObjectValidator(validateIAMFilename)
}

func validateIAMFilename(o *ast.NamespaceObject) error {
	if o.GroupVersionKind().GroupKind() == kinds.IAMPolicy() {
		filename := o.Relative.Base()
		if filename != allowedIAMFilename {
			return vet.UndocumentedErrorf(
				"IAMPolicies MUST be declared in a file named %q, but %q declares an IAMPolicy.",
				allowedIAMFilename, filename)
		}
	}
	return nil
}
