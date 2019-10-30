package filesystem

import (
	"os"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
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

			object := asDefaultVersionedOrOriginal(info.Object, info.Mapping)
			// TODO: Remove the validateAnnotations check when we migrate to
			//  apimachinery 1.17 since it is fixed then.
			errs = status.Append(errs, validateAnnotations(info.Object.(runtime.Unstructured), source.Path()))

			fileObject := ast.NewFileObject(object, source.Path())
			isNomosObject := info.Object.GetObjectKind().GroupVersionKind().Group == configmanagement.GroupName
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
		o := ast.NewFileObject(u, path)
		return invalidAnnotationValueError(&o, invalidKeys)
	}
	return nil
}

func (r *FileReader) getBuilder(stubMissing bool, crds ...*v1beta1.CustomResourceDefinition) *resource.Builder {
	return resource.NewBuilder(importer.NewFilesystemCRDAwareClientGetter(r.ClientGetter, stubMissing, crds...))
}
