package syntax

import (
	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// NewDisallowedFieldsValidator validates that imported objects do not contain disallowed fields.
func NewDisallowedFieldsValidator() *visitor.ValidatorVisitor {
	return visitor.NewAllObjectValidator(func(o ast.FileObject) status.MultiError {
		m := o.MetaObject()
		if len(m.GetOwnerReferences()) > 0 {
			return status.From(vet.IllegalFieldsInConfigError(&o, id.OwnerReference))
		}
		if m.GetSelfLink() != "" {
			return status.From(vet.IllegalFieldsInConfigError(&o, id.SelfLink))
		}
		if m.GetUID() != "" {
			return status.From(vet.IllegalFieldsInConfigError(&o, id.UID))
		}
		if m.GetResourceVersion() != "" {
			return status.From(vet.IllegalFieldsInConfigError(&o, id.ResourceVersion))
		}
		if m.GetGeneration() != 0 {
			return status.From(vet.IllegalFieldsInConfigError(&o, id.Generation))
		}
		if !m.GetCreationTimestamp().Time.IsZero() {
			return status.From(vet.IllegalFieldsInConfigError(&o, id.CreationTimestamp))
		}
		if m.GetDeletionTimestamp() != nil {
			return status.From(vet.IllegalFieldsInConfigError(&o, id.DeletionTimestamp))
		}
		if m.GetDeletionGracePeriodSeconds() != nil {
			return status.From(vet.IllegalFieldsInConfigError(&o, id.DeletionGracePeriodSeconds))
		}
		if o.GroupVersionKind().Group != v1.SchemeGroupVersion.Group {
			// We don't need to check status fields for nomos resources, they are never synced.
			if u, err := o.Unstructured(); err != nil {
				return status.From(vet.ObjectParseError(&o))
			} else if hasStatusField(u) {
				return status.From(vet.IllegalFieldsInConfigError(&o, id.Status))
			}
		}

		return nil
	})
}

func hasStatusField(u runtime.Unstructured) bool {
	// The following call will only error out if the UnstructuredContent returns something that is not a map.
	// This has already been verified upstream.
	m, ok, err := unstructured.NestedFieldNoCopy(u.UnstructuredContent(), "status")
	if err != nil {
		// This should never happen!!!
		glog.Errorf("unexpected error retrieving status field: %v:\n%v", err, u)
	}
	return ok && m != nil
}
