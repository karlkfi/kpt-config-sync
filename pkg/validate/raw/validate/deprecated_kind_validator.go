package validate

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// invalidToValidGroupKinds is a mapping from deprecated GroupKinds to the
// current version of the GroupKind that the config in the repo should be
// replaced with.
var invalidToValidGroupKinds = map[schema.GroupKind]schema.GroupVersionKind{
	v1beta1.SchemeGroupVersion.WithKind("DaemonSet").GroupKind():         kinds.DaemonSet(),
	v1beta1.SchemeGroupVersion.WithKind("Deployment").GroupKind():        kinds.Deployment(),
	v1beta1.SchemeGroupVersion.WithKind("Ingress").GroupKind():           kinds.Ingress(),
	v1beta1.SchemeGroupVersion.WithKind("ReplicaSet").GroupKind():        kinds.ReplicaSet(),
	v1beta1.SchemeGroupVersion.WithKind("NetworkPolicy").GroupKind():     kinds.NetworkPolicy(),
	v1beta1.SchemeGroupVersion.WithKind("PodSecurityPolicy").GroupKind(): kinds.PodSecurityPolicy(),
	v1beta1.SchemeGroupVersion.WithKind("StatefulSet").GroupKind():       kinds.StatefulSet(),
}

// DeprecatedKinds verifies that the given FileObject is not deprecated.
func DeprecatedKinds(obj ast.FileObject) status.Error {
	gk := obj.GetObjectKind().GroupVersionKind().GroupKind()
	if expected, invalid := invalidToValidGroupKinds[gk]; invalid {
		return nonhierarchical.DeprecatedGroupKindError(obj, expected)
	}
	return nil
}
