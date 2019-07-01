package hydrate

import (
	"fmt"
	"strings"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ToFileObjects sets a default file path for each object, guaranteed to be unique for a collection
// of runtime.Objects which do not collide (group/kind/namespace/name duplication)
func ToFileObjects(extension string, objects ...runtime.Object) []ast.FileObject {
	result := make([]ast.FileObject, len(objects))
	duplicates := make(map[string]int, len(objects))
	for i, obj := range objects {
		fo := ast.NewFileObject(obj, cmpath.FromSlash(defaultPath(extension, obj)))
		result[i] = fo
		duplicates[fo.SlashPath()]++
	}

	for i, obj := range result {
		if duplicates[obj.SlashPath()] > 1 {
			result[i] = ast.NewFileObject(obj.Object, cmpath.FromSlash(longPath(extension, obj.Object)))
		}
	}

	return result
}

// defaultPath returns the default (short) path in the repository.
func defaultPath(extension string, o runtime.Object) string {
	metaObj := o.(metav1.Object)
	gvk := o.GetObjectKind().GroupVersionKind()
	path := fmt.Sprintf("%s_%s.%s", gvk.Kind, metaObj.GetName(), extension)
	path = strings.ToLower(path)
	if namespace := metaObj.GetNamespace(); namespace != "" {
		path = fmt.Sprintf("%s/%s", namespace, path)
	}
	return path
}

// longPath returns the long path which ensures guarantees unique filenames for a valid set of
// manifests (no group/kind/name collisions).
func longPath(extension string, o runtime.Object) string {
	metaObj := o.(metav1.Object)
	gvk := o.GetObjectKind().GroupVersionKind()
	path := fmt.Sprintf("%s.%s_%s.%s", gvk.Kind, gvk.Group, metaObj.GetName(), extension)
	path = strings.ToLower(path)
	if namespace := metaObj.GetNamespace(); namespace != "" {
		path = fmt.Sprintf("%s/%s", namespace, path)
	}
	return path
}
