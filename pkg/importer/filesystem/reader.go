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
		return nil, status.From(status.PathWrapf(err, dir.AbsoluteOSPath()))
	}

	// We do this visitor length check because Builder returns an untyped error if passed an empty
	// directory. There is no way via the library to disable it, so this logic is effectively copy-
	// pasted out.
	visitors, err := resource.ExpandPathsToFileVisitors(
		nil, dir.AbsoluteOSPath(), true, resource.FileExtensions, nil)
	if err != nil {
		return nil, status.From(status.PathWrapf(err, dir.AbsoluteOSPath()))
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
			return nil, status.From(status.APIServerWrapf(err, "failed to get resource infos"))
		}
		for _, info := range fileInfos {
			//Assign relative path since that's what we actually need.
			source, err := dir.Root().Rel(cmpath.FromOS(info.Source))
			if err != nil {
				errs = status.Append(errs, err)
				continue
			}

			object := asDefaultVersionedOrOriginal(info.Object, info.Mapping)
			fileObject := ast.NewFileObject(object, source.Path())
			isNomosObject := info.Object.GetObjectKind().GroupVersionKind().Group == configmanagement.GroupName
			if !isNomosObject && hasStatusField(info.Object.(runtime.Unstructured)) {
				errs = status.Append(errs, status.From(syntax.IllegalFieldsInConfigError(&fileObject, id.Status)))
			}
			fileObjects = append(fileObjects, fileObject)
		}
	}
	return fileObjects, errs
}

func (r *FileReader) getBuilder(stubMissing bool, crds ...*v1beta1.CustomResourceDefinition) *resource.Builder {
	return resource.NewBuilder(importer.NewFilesystemCRDAwareClientGetter(r.ClientGetter, stubMissing, crds...))
}
