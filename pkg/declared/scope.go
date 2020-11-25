package declared

import (
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/util/validation"
)

// Scope defines a distinct (but not necessarily disjoint) area of responsibility
// for a Reconciler.
type Scope string

// RootReconciler is a special constant for a Scope for a reconciler which is
// running as the "root reconciler" (vs a namespace reconciler).
//
// This Scope takes precedence over all others.
const RootReconciler = Scope(":root")

// ValidateScope ensures the passed string is either the special RootReconciler value
// or is a valid Namespace name.
func ValidateScope(s string) error {
	if s == string(RootReconciler) {
		return nil
	}
	errs := validation.IsDNS1123Subdomain(s)
	if len(errs) > 0 {
		return status.InternalErrorf("invalid scope %q: %v", s, errs)
	}
	return nil
}

// ScopeName returns the RootSync CR name if the passed Scope is the Root,
// otherwise it returns the RepoSync CR name.
func ScopeName(scope Scope) string {
	if scope == RootReconciler {
		return v1alpha1.RootSyncName
	}
	return v1alpha1.RepoSyncName
}
