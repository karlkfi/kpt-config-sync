package syntax

import (
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IllegalFieldsInConfigErrorCode is the error code for IllegalFieldsInConfigError
const IllegalFieldsInConfigErrorCode = "1045"

type hasSelfLink interface {
	GetSelfLink() string
}

type hasGeneration interface {
	GetGeneration() int64
}

type hasCreationTimestamp interface {
	GetCreationTimestamp() v1.Time
}

type hasDeletionTimestamp interface {
	GetDeletionTimestamp() *v1.Time
}

type hasDeletionGracePeriodSeconds interface {
	GetDeletionGracePeriodSeconds() *int64
}

// DisallowFields returns an error if o contains any disallowed fields.
func DisallowFields(o ast.FileObject) status.Error {
	obj := o.Object
	if refs, ok := obj.(core.OwnerReferenced); ok {
		if len(refs.GetOwnerReferences()) > 0 {
			return IllegalFieldsInConfigError(o, id.OwnerReference)
		}
	}
	if link, ok := obj.(hasSelfLink); ok {
		if link.GetSelfLink() != "" {
			return IllegalFieldsInConfigError(o, id.SelfLink)
		}
	}
	if o.GetUID() != "" {
		return IllegalFieldsInConfigError(o, id.UID)
	}
	if o.GetResourceVersion() != "" {
		return IllegalFieldsInConfigError(o, id.ResourceVersion)
	}
	if gen, ok := obj.(hasGeneration); ok {
		if gen.GetGeneration() != 0 {
			return IllegalFieldsInConfigError(o, id.Generation)
		}
	}
	if creation, ok := obj.(hasCreationTimestamp); ok {
		if !creation.GetCreationTimestamp().Time.IsZero() {
			return IllegalFieldsInConfigError(o, id.CreationTimestamp)
		}
	}
	if deletion, ok := obj.(hasDeletionTimestamp); ok {
		if deletion.GetDeletionTimestamp() != nil {
			return IllegalFieldsInConfigError(o, id.DeletionTimestamp)
		}
	}
	if gracePeriod, ok := obj.(hasDeletionGracePeriodSeconds); ok {
		if gracePeriod.GetDeletionGracePeriodSeconds() != nil {
			return IllegalFieldsInConfigError(o, id.DeletionGracePeriodSeconds)
		}
	}
	return nil
}

var illegalFieldsInConfigErrorBuilder = status.NewErrorBuilder(IllegalFieldsInConfigErrorCode)

// IllegalFieldsInConfigError reports that an object has an illegal field set.
func IllegalFieldsInConfigError(resource id.Resource, field id.DisallowedField) status.Error {
	return illegalFieldsInConfigErrorBuilder.
		Sprintf("Configs with %[1]q specified are not allowed. "+
			"To fix, either remove the config or remove the %[1]q field in the config:",
			field).
		BuildWithResources(resource)
}
