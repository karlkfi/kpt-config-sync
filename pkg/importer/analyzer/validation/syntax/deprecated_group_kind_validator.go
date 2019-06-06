package syntax

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
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
}

// NewDeprecatedGroupKindValidator returns a Visitor that checks for deprecated config GroupKinds.
func NewDeprecatedGroupKindValidator() *visitor.ValidatorVisitor {
	return visitor.NewAllObjectValidator(func(o ast.FileObject) status.MultiError {
		gk := o.GroupVersionKind().GroupKind()
		if expected, invalid := invalidToValidGroupKinds[gk]; invalid {
			return status.From(vet.DeprecatedGroupKindError(&o, expected))
		}
		return nil
	})
}
