package syntax

import (
	"path"

	"github.com/google/nomos/pkg/api/policyhierarchy"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilvalidation "k8s.io/apimachinery/pkg/util/validation"
)

var (
	// NamespaceGVK is the GroupVersionKind of the canonical Namespace Kind
	NamespaceGVK = corev1.SchemeGroupVersion.WithKind("Namespace")
)

// MetadataNameValidator validates the value of metadata.name
var MetadataNameValidator = &FileObjectValidator{
	validate: func(fileObject ast.FileObject) error {
		gvk := fileObject.GroupVersionKind()

		if fileObject.Name() == "" {
			// Name MUST NOT be empty
			return vet.MissingObjectNameError{Object: fileObject}
		} else if isDefaultCrdAllowedInNomos(gvk) {
			// If CRD, then namee must be a valid DNS1123 subdomain
			errs := utilvalidation.IsDNS1123Subdomain(fileObject.Name())
			if errs != nil {
				return vet.InvalidMetadataNameError{Object: fileObject}
			}
		} else if gvk == NamespaceGVK {
			expectedName := path.Base(path.Dir(fileObject.Source))
			if expectedName == repo.NamespacesDir {
				return vet.IllegalTopLevelNamespaceError{Object: fileObject}
			}
			if fileObject.Name() != expectedName {
				return vet.InvalidNamespaceNameError{Source: fileObject.Source, Expected: expectedName, Actual: fileObject.Name()}
			}
		}
		return nil
	},
}

// isDefaultCrdAllowedInNomos checks if a Resource is a CRD that comes with a default Nomos installation.
//
// This explicitly does not check for Nomos or Application even though they are CRDs because they
// should never be in a Nomos repository anyway.
func isDefaultCrdAllowedInNomos(gvk schema.GroupVersionKind) bool {
	return gvk.Group == policyhierarchy.GroupName ||
		(gvk.Group == "clusterregistry.k8s.io" && gvk.Kind == "Cluster")
}
