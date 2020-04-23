package filesystem

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/validation/syntax"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// Reader reads a list of FileObjects.
type Reader interface {
	// Read returns the list of FileObjects in the passed directory.
	//
	// dir is either a repository root directory, or a path relative to one.
	// Returned fileObjects have their path set to be relative to the repository
	// root.
	Read(dir cmpath.RootedPath) ([]ast.FileObject, status.MultiError)
}

// FileReader reads FileObjects from a filesystem.
type FileReader struct{}

var _ Reader = &FileReader{}

func evalSymlinks(dir string) (string, error) {
	p, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return "", status.PathWrapError(err, dir)
	}
	// Symlinks can be relative paths, so force the path to be absolute.
	p, err = filepath.Abs(p)
	if err != nil {
		return "", status.PathWrapError(err, dir)
	}
	return p, nil
}

// Read implements Reader.
func (r *FileReader) Read(dir cmpath.RootedPath) ([]ast.FileObject, status.MultiError) {
	if _, err := os.Stat(dir.AbsoluteOSPath()); os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, status.PathWrapError(err, dir.AbsoluteOSPath())
	}

	rootDir, err := evalSymlinks(dir.Root().AbsoluteOSPath())
	if err != nil {
		return nil, status.PathWrapError(err, dir.AbsoluteOSPath())
	}
	p, err := evalSymlinks(dir.AbsoluteOSPath())
	if err != nil {
		return nil, status.PathWrapError(err, dir.AbsoluteOSPath())
	}

	var errs status.MultiError
	var fileObjects []ast.FileObject
	walkErr := filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			errs = status.Append(errs, status.PathWrapError(err, path))
			return nil
		}

		if info.IsDir() {
			// This is a directory, continue.
			return nil
		}

		// Read the file to a list of unstructured objects.
		unstructureds, err := parseFile(path)
		if err != nil {
			errs = status.Append(errs, status.PathWrapError(err, path))
			return nil
		}

		// Assign relative path since that's what we actually need.
		source, relErr := filepath.Rel(rootDir, path)
		if relErr != nil {
			// We couldn't get the relative path from the repository root. Something is very wrong.
			errs = status.Append(errs, relErr)
		}

		for _, u := range unstructureds {
			newFileObjects, err := toFileObjects(u, cmpath.FromOS(source))
			if err != nil {
				errs = status.Append(errs, err)
			}
			fileObjects = append(fileObjects, newFileObjects...)
		}

		return nil
	})
	if walkErr != nil {
		return nil, status.PathWrapError(walkErr, dir.AbsoluteOSPath())
	}

	return fileObjects, errs
}

// toFileObjects returns either:
// 1) a slice containing a single FileObject, if the passed Unstructured is a valid Kubernetes object;
// 2) a list of multiple FileObject, if the passed Unstructured was a List type; or
// 3) an error, if there was a problem parsing the Unstructured.
func toFileObjects(unstructured runtime.Unstructured, path cmpath.Path) ([]ast.FileObject, status.MultiError) {
	if isList(unstructured) {
		return flattenList(unstructured, path)
	}

	isNomosObject := unstructured.GetObjectKind().GroupVersionKind().Group == configmanagement.GroupName
	if !isNomosObject && hasStatusField(unstructured) {
		return nil, syntax.IllegalFieldsInConfigError(&unstructuredID{Unstructured: unstructured}, id.Status)
	}

	// TODO: Remove the validateMetadata check when we migrate to
	//  apimachinery 1.17 since it is fixed then.
	err := validateMetadata(unstructured, path)
	if err != nil {
		return nil, err
	}

	obj, ok := asDefaultVersionedOrOriginal(unstructured).(core.Object)
	if !ok {
		// The type doesn't declare required fields, but is registered.
		// User-specified types are implicitly Unstructured, which defines Labels/Annotations/etc. even
		// if the underlying type definition does _NOT_. It isn't clear how this code would ever be reached.
		return nil, status.InternalErrorf("not a valid persistent Kubernetes type: %s", obj.GroupVersionKind().String())
	}
	return []ast.FileObject{ast.NewFileObject(obj, path)}, nil
}

func flattenList(list runtime.Unstructured, path cmpath.Path) ([]ast.FileObject, status.MultiError) {
	var result []ast.FileObject
	var errs status.MultiError

	err := list.EachListItem(func(object runtime.Object) error {
		unstructuredItem, isUnstructured := object.(runtime.Unstructured)
		if !isUnstructured {
			// It isn't clear how this would happen, as by default objects are parsed to runtime.Unstructured.
			errs = status.Append(errs, status.InternalErrorf("converted %s from runtime.Unstructured too soon", unstructuredItem.GetObjectKind().GroupVersionKind().String()))
			return nil
		}
		// If the unstructuredItem is itself a List, toFileObjects recurse back here until we get to an actual Object.
		newObjs, newErrs := toFileObjects(unstructuredItem, path)
		result = append(result, newObjs...)
		errs = status.Append(errs, newErrs)
		// Returning an error will stop parsing early, so return nil.
		return nil
	})
	errs = status.Append(errs, err)
	return result, errs
}

// isList detects whether the runtime.Unstructured is actually a List.
func isList(uList runtime.Unstructured) bool {
	if uList.IsList() {
		// IsList works for registered List types only.
		// It fails to work properly for nested lists and List types we haven't registered in scheme.Scheme.
		return true
	}
	// The runtime.Unstructured API claims that if IsList returns false then EachListItem will fail.
	// This isn't true in the case where the type defines ListInterface, so we can safely
	// use this for nested Lists even though IsList returns false, assuming it meets the below criteria.

	if !strings.HasSuffix(uList.GetObjectKind().GroupVersionKind().Kind, "List") {
		// The name of a List kind MUST end in List, per the Kubernetes API conventions.
		// Thus, if the suffix is missing we know this cannot be a List.
		return false
	}

	// Parse the object into memory. If it is a List type, it MUST match the ListInterface.
	listObj := asDefaultVersionedOrOriginal(uList)
	_, isList := listObj.(metav1.ListInterface)
	return isList
}

// asDefaultVersionedOrOriginal returns the object as a Go object in the external form.
// If the GVK is registered in scheme.Scheme, return that version. Otherwise, try to return the declared version.
// If this fails, returns the original runtime.Unstructured.
func asDefaultVersionedOrOriginal(obj runtime.Unstructured) runtime.Object {
	// Determine the GroupVersion to convert the object to.
	groupVersioner := runtime.GroupVersioner(schema.GroupVersions(scheme.Scheme.PrioritizedVersionsAllGroups()))
	if _, ok := groupVersioner.KindForGroupVersionKinds([]schema.GroupVersionKind{obj.GetObjectKind().GroupVersionKind()}); !ok {
		// If the scheme doesn't have the GVK, try to serialize to the declared GV.
		groupVersioner = obj.GetObjectKind().GroupVersionKind().GroupVersion()
	}

	converter := runtime.ObjectConvertor(scheme.Scheme)
	if cObj, err := converter.ConvertToVersion(obj, groupVersioner); err == nil {
		return cObj
	}

	return obj
}

// validateMetadata returns a status.MultiError if metadata.annotations/labels
// has a value that wasn't parsed as a string. This is a workaround the
// k8s.io/apimachinery bug that causes the entire annotations field to be
// silently discarded if any values aren't strings.
//
// TODO(b/154838005): Remove this once we upgrade and don't need to explicitly
//  check for this.
func validateMetadata(u runtime.Unstructured, path cmpath.Path) status.MultiError {
	content := u.UnstructuredContent()
	metadata, hasMetadata := content["metadata"].(map[string]interface{})
	if !hasMetadata {
		return status.UndocumentedError("All persistent Kubernetes objects MUST define metadata.")
	}

	annotations, hasAnnotations := metadata["annotations"]
	labels, hasLabels := metadata["labels"]
	if !hasAnnotations && !hasLabels {
		// No annotations or labels, so nothing to validate.
		return nil
	}

	var errs status.MultiError
	invalidAnnotations, err := getInvalidKeys(annotations)
	errs = status.Append(errs, err)
	if len(invalidAnnotations) > 0 {
		errs = status.Append(errs, InvalidAnnotationValueError(&unstructuredID{Unstructured: u, Path: path}, invalidAnnotations))
	}
	invalidLabels, err := getInvalidKeys(labels)
	errs = status.Append(errs, err)
	if len(invalidLabels) > 0 {
		errs = status.Append(errs, InvalidLabelValueError(&unstructuredID{Unstructured: u, Path: path}, invalidLabels))
	}

	return errs
}

func getInvalidKeys(o interface{}) ([]string, status.MultiError) {
	if o == nil {
		return nil, nil
	}
	m, isMap := o.(map[string]interface{})
	if !isMap {
		// We don't expect this error to be thrown since the parser before it would
		// already return an error. Thus, creating a type just for this case would
		// be overkill.
		return nil, status.UndocumentedError("metadata.labels and metadata.annotations must be a map")
	}

	var result []string
	for key, value := range m {
		if _, isString := value.(string); !isString {
			// The value wasn't parsed as a string.
			result = append(result, key)
		}
	}
	return result, nil
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
	//  This occurs for any type with ListMeta, and for non-persisted types
	return ""
}

// GroupVersionKind implements id.Resource.
func (u unstructuredID) GroupVersionKind() schema.GroupVersionKind {
	return u.Unstructured.GetObjectKind().GroupVersionKind()
}
