package vet

import (
	"strings"

	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// MetadataNameCollisionErrorCode is the error code for ObjectNameCollisionError
const MetadataNameCollisionErrorCode = "1029"

func init() {
	r1 := role()
	r1.Path = cmpath.FromSlash("namespaces/foo/r1.yaml")
	r2 := role()
	r2.Path = cmpath.FromSlash("namespaces/foo/r2.yaml")
	status.AddExamples(MetadataNameCollisionErrorCode, MetadataNameCollisionError(
		r1, r2,
	))
}

var metadataNameCollisionError = status.NewErrorBuilder(MetadataNameCollisionErrorCode)

// MetadataNameCollisionError reports that multiple objects in the same namespace of the same Kind share a name.
func MetadataNameCollisionError(resources ...id.Resource) status.Error {
	return metadataNameCollisionError.WithResources(resources...).Errorf(
		"Configs of the same Kind MUST have unique names in the same %[1]s and their parent %[2]ss:",
		node.Namespace, strings.ToLower(string(node.AbstractNamespace)))
}
