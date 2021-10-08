package hydrate

import (
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/differ"
	"github.com/google/nomos/pkg/validate/objects"
	"sigs.k8s.io/cli-utils/pkg/common"
)

// PreventDeletion adds the `client.lifecycle.config.k8s.io/deletion: detach` annotation to special namespaces,
// which include `default`, `kube-system`, `kube-public`, `kube-node-lease`, and `gatekeeper-system`.
func PreventDeletion(objs *objects.Raw) status.MultiError {
	for _, obj := range objs.Objects {
		if obj.GetObjectKind().GroupVersionKind().GroupKind() == kinds.Namespace().GroupKind() && differ.SpecialNamespaces[obj.GetName()] {
			core.SetAnnotation(obj, common.LifecycleDeleteAnnotation, common.PreventDeletion)
		}
	}
	return nil
}
