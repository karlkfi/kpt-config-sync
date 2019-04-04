package differ

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	corev1 "k8s.io/api/core/v1"
)

// NamespaceDiff represents a diff between a Namespace config and the one on the cluster.
type NamespaceDiff struct {
	Name     string
	Declared *v1.NamespaceConfig
	Actual   *corev1.Namespace
}

// Type returns the type of the NamespaceDiff.
// TODO(willbeason): Merge NamespaceDiff with Diff since there's overlap.
func (d *NamespaceDiff) Type() Type {

	if d.Declared != nil {
		// The NamespaceConfig IS in the repository.
		if managementUnset(d.Declared) {
			// The declared Namespace has no resource management key, so it is managed.
			if d.Actual != nil {
				// The Namespace is also in the cluster, so update it.
				return Update
			}
			// The Namespace is not in the cluster, so create it.
			return Create
		}
		if managementDisabled(d.Declared) {
			// The Namespace is explicitly marked management disabled in the repository.
			if d.Actual != nil {
				if hasNomosMeta(d.Actual) {
					// Management is disabled for the Namespace, so remove management annotations from the API Server.
					return Unmanage
				}
			}
			// Management disabled and there's no required changes to the Namespace.
			return NoOp
		}
		// The management annotation in the repo is invalid, so show an error.
		return Error
	}

	// The Namespace IS NOT in the repository.
	if d.Actual != nil {
		// The Namespace IS on the API Server.
		if !hasNomosMeta(d.Actual) {
			// No Nomos annotations or labels, so don't do anything.
			return NoOp
		}

		// There are Nomos annotations or labels on the Namespace.
		if managementEnabled(d.Actual) {
			// Delete Namespace with management enabled on API Server.
			return Delete
		}
		// The Namespace has Nomos artifacts but is unmanaged, so remove them.
		return Unmanage
	}

	// The Namespace does not exist on the API Server and has no corresponding NamespaceConfig, so do nothing.
	return NoOp
}
