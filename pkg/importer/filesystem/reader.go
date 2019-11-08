package filesystem

import (
	"os"

	"github.com/google/nomos/pkg/core"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericclioptions/resource"

	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/importer"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// Reader reads a list of FileObjects.
type Reader interface {
	// Read returns the list of FileObjects in the passed directory.
	//
	// stubMissing disables the need for CRDs to be present in order to parse unknown objects.
	Read(dir cmpath.Relative, stubMissing bool, crds ...*v1beta1.CustomResourceDefinition) ([]ast.FileObject, status.MultiError)
}

// FileReader reads FileObjects from a filesystem.
type FileReader struct {
	ClientGetter genericclioptions.RESTClientGetter
}

var _ Reader = &FileReader{}

func (r *FileReader) Read(dir cmpath.Relative, stubMissing bool, crds ...*v1beta1.CustomResourceDefinition) ([]ast.FileObject, status.MultiError) {
	if _, err := os.Stat(dir.AbsoluteOSPath()); os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, status.PathWrapf(err, dir.AbsoluteOSPath())
	}

	// We do this visitor length check because Builder returns an untyped error if passed an empty
	// directory. There is no way via the library to disable it, so this logic is effectively copy-
	// pasted out.
	visitors, err := resource.ExpandPathsToFileVisitors(
		nil, dir.AbsoluteOSPath(), true, resource.FileExtensions, nil)
	if err != nil {
		return nil, status.PathWrapf(err, dir.AbsoluteOSPath())
	}

	var errs status.MultiError
	var fileObjects []ast.FileObject
	if len(visitors) > 0 {
		options := &resource.FilenameOptions{Recursive: true, Filenames: []string{dir.AbsoluteOSPath()}}
		builder := r.getBuilder(stubMissing, crds...)
		result := builder.
			Unstructured().
			ContinueOnError().
			FilenameParam(false, options).
			Do()
		fileInfos, err := result.Infos()
		if err != nil {
			return nil, status.APIServerWrapf(err, "failed to get resource infos")
		}
		for _, info := range fileInfos {
			//Assign relative path since that's what we actually need.
			source, err := dir.Root().Rel(cmpath.FromOS(info.Source))
			if err != nil {
				errs = status.Append(errs, err)
				continue
			}

			// TODO(143557906): Don't assume this is an Object as it might be a List.
			obj := asDefaultVersionedOrOriginal(info.Object, info.Mapping).(core.Object)

			// TODO: Remove the validateAnnotations check when we migrate to
			//  apimachinery 1.17 since it is fixed then.
			errs = status.Append(errs, validateAnnotations(info.Object.(runtime.Unstructured), source.Path()))

			fileObject := ast.NewFileObject(obj, source.Path())
			isNomosObject := fileObject.GroupVersionKind().Group == configmanagement.GroupName
			if !isNomosObject && hasStatusField(info.Object.(runtime.Unstructured)) {
				errs = status.Append(errs, syntax.IllegalFieldsInConfigError(&fileObject, id.Status))
			}
			fileObjects = append(fileObjects, fileObject)
		}
	}
	return fileObjects, errs
}

// validateAnnotations returns a status.MultiError if metadata.annotations
// has a value that wasn't parsed as a string. This is a workaround the
// k8s.io/apimachinery bug that causes the entire annotations field to be
// silently discarded if any values aren't strings.
func validateAnnotations(u runtime.Unstructured, path cmpath.Path) status.MultiError {
	content := u.UnstructuredContent()
	metadata := content["metadata"].(map[string]interface{})
	annotations, hasAnnotations := metadata["annotations"]
	if !hasAnnotations {
		// No annotations, so nothing to validate.
		return nil
	}
	annotationsMap, isMap := annotations.(map[string]interface{})
	if !isMap {
		// We don't expect this error to be thrown since the parser before it would
		// already return an error. Thus, creating a type just for this case would
		// be overkill.
		return status.UndocumentedError.New("metadata.annotations must be a map")
	}

	var invalidKeys []string
	for key, value := range annotationsMap {
		if _, isString := value.(string); !isString {
			// The value wasn't parsed as a string.
			invalidKeys = append(invalidKeys, key)
		}
	}
	if invalidKeys != nil {
		return invalidAnnotationValueError(&unstructuredID{Unstructured: u, Path: path}, invalidKeys)
	}
	return nil
}

type unstructuredID struct {
	runtime.Unstructured
	cmpath.Path
}

var _ id.Resource = &unstructuredID{}

// GetNamespace implements id.Resource.
func (u unstructuredID) GetNamespace() string {
	if namespaced, isNamespaced := u.Unstructured.(core.Namespaced); isNamespaced {
		return namespaced.GetNamespace()
	}
	// We already hide Namespace from messages if it is empty string, so we don't have to handle this case specially.
	return ""
}

// GetName implements id.Resource.
func (u unstructuredID) GetName() string {
	if named, isNamed := u.Unstructured.(core.Named); isNamed {
		return named.GetName()
	}
	// TODO(143557906): Handle displaying errors for type that don't define metadata.name.
	//  This occurs for any type with ListMeta.
	return ""
}

// GroupVersionKind implements id.Resource.
func (u unstructuredID) GroupVersionKind() schema.GroupVersionKind {
	return u.Unstructured.GetObjectKind().GroupVersionKind()
}

func (r *FileReader) getBuilder(stubMissing bool, crds ...*v1beta1.CustomResourceDefinition) *resource.Builder {
	return resource.NewBuilder(importer.NewFilesystemCRDAwareClientGetter(r.ClientGetter, stubMissing, crds...))
}
