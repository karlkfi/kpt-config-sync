package hierarchical

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/parsed"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// SingletonValidator returns a visitor which ensures the Root contains no more
// than one of the passed GroupVersionKind.
func SingletonValidator(gvk schema.GroupVersionKind) parsed.ValidatorFunc {
	return parsed.ValidateAllObjects(func(objs []ast.FileObject) status.MultiError {
		return validateSingleton(objs, gvk)
	})
}

// TreeNodeSingletonValidator returns a visitor which ensures every TreeNode has
// at most one of the passed GroupVersionKind.
func TreeNodeSingletonValidator(gvk schema.GroupVersionKind) parsed.ValidatorFunc {
	// A TreeRoot will call this with all of the FileObjects for each TreeNode,
	// one at a time.
	return parsed.ValidateNamespaceObjects(func(objs []ast.FileObject) status.MultiError {
		return validateSingleton(objs, gvk)
	})
}

func validateSingleton(objs []ast.FileObject, gvk schema.GroupVersionKind) status.Error {
	var duplicates []id.Resource
	for _, object := range objs {
		if object.GroupVersionKind() == gvk {
			duplicates = append(duplicates, object)
		}
	}
	if len(duplicates) > 1 {
		return status.MultipleSingletonsError(duplicates...)
	}
	return nil
}
