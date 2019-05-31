package gcpconfig

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NewFilenameValidator creates a new validator that ensures objects under
// GCP policy management are only declared in files with a particular name.
func NewFilenameValidator() *visitor.ValidatorVisitor {
	return visitor.NewObjectValidator(validateFilename)
}

var allowedGroupKindToFileName = map[schema.GroupKind]string{
	kinds.Organization():       "gcp-organization.yaml",
	kinds.Folder():             "gcp-folder.yaml",
	kinds.Project():            "gcp-project.yaml",
	kinds.IAMPolicy():          "gcp-iam-policy.yaml",
	kinds.OrganizationPolicy(): "gcp-organization-policy.yaml",
}

func validateFilename(o *ast.NamespaceObject) status.MultiError {
	gk := o.GroupVersionKind().GroupKind()
	if allowed, ok := allowedGroupKindToFileName[gk]; ok {
		filename := o.Base()
		if filename != allowed {
			return status.From(vet.InvalidGcpResourceFilenameError(o.Path))
		}
	}
	return nil
}
