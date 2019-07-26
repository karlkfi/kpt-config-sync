package vet

import (
	"fmt"
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
	status.AddExamples(MetadataNameCollisionErrorCode, NamespaceMetadataNameCollisionError(
		r1, r2,
	))
}

var metadataNameCollisionErrorBuilder = status.NewErrorBuilder(MetadataNameCollisionErrorCode)

// NamespaceMetadataNameCollisionError reports that multiple namespace-scoped objects of the same Kind and
// namespace have the same metadata name
func NamespaceMetadataNameCollisionError(resources ...id.Resource) status.Error {
	return metadataNameCollisionErrorBuilder.WithResources(resources...).Errorf(
		fmt.Sprintf("Namespace configs of the same Kind MUST have unique names if they also have the same %[1]s or parent %[2]s(s):",
			node.Namespace, strings.ToLower(string(node.AbstractNamespace))))
}

// ClusterMetadataNameCollisionError reports that multiple cluster-scoped objects of the same Kind and
// namespace have the same metadata name
func ClusterMetadataNameCollisionError(resources ...id.Resource) status.Error {
	return metadataNameCollisionErrorBuilder.WithResources(resources...).New(
		"Cluster configs of the same Kind MUST have unique names")
}
