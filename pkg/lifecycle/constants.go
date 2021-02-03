// Package lifecycle defines the client-side lifecycle directives ACM honors.
//
// Implementation conforms with:
// go/lifecycle-directives-in-detail
package lifecycle

import (
	"github.com/google/nomos/pkg/core"
	"sigs.k8s.io/cli-utils/pkg/common"
)

// HasPreventDeletion returns true if the object has the LifecycleDeleteAnnotation
// and it is set to "detach".
func HasPreventDeletion(o core.Object) bool {
	deletion, hasDeletion := o.GetAnnotations()[common.LifecycleDeleteAnnotation]
	return hasDeletion && (deletion == common.PreventDeletion)
}
