package vet

import (
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
)

// InvalidGcpResourceFilenameErrorCode is the error code for InvalidGcpResourceFilenameError
const InvalidGcpResourceFilenameErrorCode = "1051"

func init() {
	status.AddExamples(InvalidGcpResourceFilenameErrorCode, InvalidGcpResourceFilenameError(cmpath.FromSlash("/usr/local/home/bob/grepo/hierarchy/bob-organization/gcp-invalid-resource.yaml")))
}

var invalidGcpResourceFilenameError = status.NewErrorBuilder(InvalidGcpResourceFilenameErrorCode)

// InvalidGcpResourceFilenameError reports invalid GCP resource filename.
func InvalidGcpResourceFilenameError(filepath cmpath.Path) status.Error {
	// TODO(b/134175210) Need better wording on error message.
	return invalidGcpResourceFilenameError.WithPaths(filepath).New("Only these file names are supported: gcp-organization.yaml, gcp-folder.yaml, gcp-project.yaml, gcp-iam-policy.yaml, gcp-organization-policy.yaml")
}
