package nonhierarchical

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// invalidToValidGroupKinds is a mapping from deprecated GroupKinds to the current version of the GroupKind that the
// config in the repo should be replaced with.
var invalidToValidGroupKinds = map[schema.GroupKind]schema.GroupVersionKind{
	v1beta1.SchemeGroupVersion.WithKind("Deployment").GroupKind():        kinds.Deployment(),
	v1beta1.SchemeGroupVersion.WithKind("ReplicaSet").GroupKind():        kinds.ReplicaSet(),
	v1beta1.SchemeGroupVersion.WithKind("DaemonSet").GroupKind():         kinds.DaemonSet(),
	v1beta1.SchemeGroupVersion.WithKind("NetworkPolicy").GroupKind():     kinds.NetworkPolicy(),
	v1beta1.SchemeGroupVersion.WithKind("PodSecurityPolicy").GroupKind(): kinds.PodSecurityPolicy(),
	v1beta1.SchemeGroupVersion.WithKind("StatefulSet").GroupKind():       kinds.StatefulSet(),
}

// DeprecatedGroupKindValidator checks for deprecated config GroupKinds.
var DeprecatedGroupKindValidator = PerObjectValidator(deprecatedGroupKind)

func deprecatedGroupKind(o ast.FileObject) status.Error {
	gk := o.GroupVersionKind().GroupKind()
	if expected, invalid := invalidToValidGroupKinds[gk]; invalid {
		return DeprecatedGroupKindError(&o, expected)
	}
	return nil
}

// DeprecatedGroupKindErrorCode is the error code for DeprecatedGroupKindError.
const DeprecatedGroupKindErrorCode = "1050"

var deprecatedGroupKindError = status.NewErrorBuilder(DeprecatedGroupKindErrorCode)

// DeprecatedGroupKindError reports usage of a deprecated version of a specific Group/Kind.
func DeprecatedGroupKindError(resource id.Resource, expected schema.GroupVersionKind) status.Error {
	apiVersion, kind := expected.ToAPIVersionAndKind()
	return deprecatedGroupKindError.
		Sprintf("The config is using an unsupported Group and Kind. To fix, set the apiVersion to %q and kind to %q.",
			apiVersion, kind).
		BuildWithResources(resource)
}
