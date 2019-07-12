package hydrate

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ToFileObjects sets a default file path for each object, guaranteed to be unique for a collection
// of runtime.Objects which do not collide (group/kind/namespace/name duplication)
func ToFileObjects(extension string, multiCluster bool, objects ...runtime.Object) []ast.FileObject {
	result := make([]ast.FileObject, len(objects))
	duplicates := make(map[string]int, len(objects))
	for i, obj := range objects {
		fo := ast.NewFileObject(obj, cmpath.FromSlash(filename(extension, obj, multiCluster, false)))
		result[i] = fo
		duplicates[fo.SlashPath()]++
	}

	for i, obj := range result {
		if duplicates[obj.SlashPath()] > 1 {
			result[i] = ast.NewFileObject(obj.Object, cmpath.FromSlash(filename(extension, obj.Object, multiCluster, true)))
		}
	}

	return result
}

func filename(extension string, o runtime.Object, includeCluster bool, includeGroup bool) string {
	metaObj := o.(metav1.Object)
	gvk := o.GetObjectKind().GroupVersionKind()
	var path string
	if includeGroup {
		path = fmt.Sprintf("%s.%s_%s.%s", gvk.Kind, gvk.Group, metaObj.GetName(), extension)
	} else {
		path = fmt.Sprintf("%s_%s.%s", gvk.Kind, metaObj.GetName(), extension)
	}
	if namespace := metaObj.GetNamespace(); namespace != "" {
		path = filepath.Join(namespace, path)
	}
	if includeCluster {
		if clusterName, found := metaObj.GetAnnotations()[v1.ClusterNameAnnotationKey]; found {
			path = filepath.Join(clusterName, path)
		} else {
			path = filepath.Join(defaultCluster, path)
		}
	}
	return strings.ToLower(path)
}
