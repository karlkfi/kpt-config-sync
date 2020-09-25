package filesystem

import (
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var structuredKinds = map[schema.GroupKind]bool{
	kinds.Cluster().GroupKind():           true,
	kinds.ClusterSelector().GroupKind():   true,
	kinds.HierarchyConfig().GroupKind():   true,
	kinds.NamespaceSelector().GroupKind(): true,
	kinds.Repo().GroupKind():              true,
	kinds.RepoSync().GroupKind():          true,
	kinds.RootSync().GroupKind():          true,
}

// mustBeStructured returns true if the importer logic requires the given GroupKind
// to be parsed into a structured object.
func mustBeStructured(gvk schema.GroupVersionKind) bool {
	return structuredKinds[gvk.GroupKind()]
}

// ObjectParseErrorCode is the code for ObjectParseError.
const ObjectParseErrorCode = "1006"

var objectParseError = status.NewErrorBuilder(ObjectParseErrorCode)

// ObjectParseError reports that an object of known type did not match its definition, and so it was
// read in as an *unstructured.Unstructured.
func ObjectParseError(resource id.Resource, err error) status.Error {
	return objectParseError.Wrap(err).
		Sprintf("The following config could not be parsed as a %v", resource.GroupVersionKind()).
		BuildWithResources(resource)
}
