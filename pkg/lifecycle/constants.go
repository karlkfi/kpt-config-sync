// Package lifecycle defines the client-side lifecycle directives ACM honors.
//
// Implementation conforms with:
// go/lifecycle-directives-in-detail
package lifecycle

import (
	"github.com/google/nomos/pkg/core"
	"sigs.k8s.io/cli-utils/pkg/common"
)

const prefix = "client.lifecycle.config.k8s.io"

// Deletion is the directive that specifies what happens when an object is
// removed from the repository.
const Deletion = prefix + "/deletion"

// PreventDeletion specifies that the resource should NOT be removed from the
// cluster if its manifest is removed from the repository.
const PreventDeletion = common.PreventDeletion

// HasPreventDeletion returns true if the object has the PreventDeletion annotation
// and it is set to "detach".
func HasPreventDeletion(o core.Object) bool {
	deletion, hasDeletion := o.GetAnnotations()[Deletion]
	return hasDeletion && (deletion == PreventDeletion)
}
