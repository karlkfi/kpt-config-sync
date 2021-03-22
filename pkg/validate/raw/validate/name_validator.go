package validate

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/status"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/rest"
)

// Name verifies that the given FileObject has a valid name according to the
// following rules:
// - the object can not have an empty name
// - if the object is related to RBAC, its name must be a valid path segment
// - otherwise the object's name must be a valid DNS1123 subdomain
func Name(obj ast.FileObject) status.Error {
	if obj.GetName() == "" {
		return nonhierarchical.MissingObjectNameError(obj)
	}

	var errs []string
	if obj.GetObjectKind().GroupVersionKind().Group == rbacv1.SchemeGroupVersion.Group {
		// The APIServer has different metadata.name requirements for RBAC types.
		errs = rest.IsValidPathSegmentName(obj.GetName())
	} else {
		errs = validation.IsDNS1123Subdomain(obj.GetName())
	}
	if errs != nil {
		return nonhierarchical.InvalidMetadataNameError(obj)
	}

	return nil
}
