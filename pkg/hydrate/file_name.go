package hydrate

import (
	"fmt"
	"path/filepath"
	"strings"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GenerateUniqueFileNames sets a default file path for each object, guaranteed to be unique for a collection
// of ast.FileObjects which do not collide (group/kind/namespace/name duplication)
func GenerateUniqueFileNames(extension string, multiCluster bool, objects ...ast.FileObject) []ast.FileObject {
	duplicates := make(map[string]int, len(objects))
	for i := range objects {
		p := cmpath.RelativeSlash(filename(extension, objects[i], multiCluster, false))
		objects[i].Relative = p
		duplicates[p.SlashPath()]++
	}

	for i, obj := range objects {
		if duplicates[obj.SlashPath()] > 1 {
			objects[i] = ast.NewFileObject(obj.Object, cmpath.RelativeSlash(filename(extension, obj.Object, multiCluster, true)))
		}
	}

	return objects
}

func filename(extension string, o client.Object, includeCluster bool, includeGroup bool) string {
	gvk := o.GetObjectKind().GroupVersionKind()
	var path string
	if includeGroup {
		path = fmt.Sprintf("%s.%s_%s.%s", gvk.Kind, gvk.Group, o.GetName(), extension)
	} else {
		path = fmt.Sprintf("%s_%s.%s", gvk.Kind, o.GetName(), extension)
	}
	if namespace := o.GetNamespace(); namespace != "" {
		path = filepath.Join(namespace, path)
	}
	if includeCluster {
		if clusterName, found := o.GetAnnotations()[v1.ClusterNameAnnotationKey]; found {
			path = filepath.Join(clusterName, path)
		} else {
			path = filepath.Join(defaultCluster, path)
		}
	}
	return strings.ToLower(path)
}
