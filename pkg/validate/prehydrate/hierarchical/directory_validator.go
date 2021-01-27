package hierarchical

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/metadata"
	"github.com/google/nomos/pkg/importer/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/parsed"
	"k8s.io/apimachinery/pkg/util/validation"
)

// ObjectDirectoryValidator returns a visitor that ensures that all namespaced
// objects are located in a directory which matches their metadata namespace.
func ObjectDirectoryValidator() parsed.ValidatorFunc {
	return parsed.ValidateNamespaceObjects(parsed.PerObjectVisitor(validateObjectDirectory))
}

func validateObjectDirectory(obj ast.FileObject) status.Error {
	if obj.GroupVersionKind() == kinds.Namespace() {
		return nil
	}
	if obj.GetNamespace() == "" {
		// We allow the metadata.namespace field to be left empty.
		return nil
	}
	expectedNamespace := obj.Dir().Base()
	if obj.GetNamespace() != expectedNamespace {
		return metadata.IllegalMetadataNamespaceDeclarationError(&obj, expectedNamespace)
	}
	return nil
}

// NamespaceDirectoryValidator returns a visitor that ensures that all Namespace
// objects are located in a directory which matches their name.
func NamespaceDirectoryValidator() parsed.ValidatorFunc {
	return parsed.ValidateNamespaceObjects(parsed.PerObjectVisitor(validateNamespaceDirectory))
}

func validateNamespaceDirectory(obj ast.FileObject) status.Error {
	if obj.GroupVersionKind() != kinds.Namespace() {
		return nil
	}
	expectedName := obj.Dir().Base()
	if expectedName == repo.NamespacesDir {
		return metadata.IllegalTopLevelNamespaceError(&obj)
	}
	if obj.GetName() != expectedName {
		return metadata.InvalidNamespaceNameError(&obj, expectedName)
	}
	return nil
}

// DirectoryNameValidator returns a visitor that ensures that all directories
// have a name which is valid for a kubernetes namespace.
func DirectoryNameValidator() parsed.ValidatorFunc {
	return parsed.ValidateNamespaceObjects(validateDirectoryName)
}

func validateDirectoryName(objs []ast.FileObject) status.MultiError {
	if len(objs) == 0 {
		return nil
	}
	directory := objs[0].Dir().Base()
	if !isValidNamespace(directory) {
		return syntax.InvalidDirectoryNameError(objs[0].Dir())
	}
	if configmanagement.IsControllerNamespace(directory) {
		return syntax.ReservedDirectoryNameError(objs[0].Dir())
	}
	return nil
}

// isValidNamespace returns true if Kubernetes allows Namespaces with the name "name".
func isValidNamespace(name string) bool {
	// IsDNS1123Label is misleading as the Kubernetes requirements are more stringent than the specification.
	errs := validation.IsDNS1123Label(name)
	return len(errs) == 0
}
