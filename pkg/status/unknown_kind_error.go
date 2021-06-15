package status

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// UnknownKindErrorCode is the error code for UnknownObjectKindError
const UnknownKindErrorCode = "1021" // Impossible to create consistent example.

var unknownKindError = NewErrorBuilder(UnknownKindErrorCode)

var errorMsg = "No CustomResourceDefinition is defined for the type %q in the cluster.\n" +
	"Resource types that are not native Kubernetes objects must have a CustomResourceDefinition.\n\n" +
	"Config Sync will retry until a CustomResourceDefinition is defined for the type %q in the cluster."

// UnknownObjectKindError reports that an object declared in the repo does not have a definition in the cluster.
func UnknownObjectKindError(resource client.Object) Error {
	gk := resource.GetObjectKind().GroupVersionKind().GroupKind()
	return unknownKindError.Sprintf(errorMsg, gk, gk).BuildWithResources(resource)
}

// UnknownGroupKindError reports that a GroupKind is not defined on the cluster, so we can't sync it.
func UnknownGroupKindError(gk schema.GroupKind) Error {
	return unknownKindError.Sprintf(errorMsg, gk, gk).Build()
}
