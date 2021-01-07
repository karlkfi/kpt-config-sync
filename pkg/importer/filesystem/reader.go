package filesystem

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
)

// Reader reads a list of FileObjects.
type Reader interface {
	// Read returns the list of FileObjects in the passed file.
	Read(filePaths FilePaths) ([]ast.FileObject, status.MultiError)
}

// FilePaths encapsulates the list of absolute file paths to read and the absolute and relative path of the Nomos Root.
type FilePaths struct {
	// RootDir is the absolute path to policyDir.
	RootDir cmpath.Absolute

	// PolicyDir is the relative path of the Nomos Root from the Repo Root.
	PolicyDir cmpath.Relative

	// Files is the list of absolute path to the files to read.
	Files []cmpath.Absolute
}

// FileReader reads FileObjects from a filesystem.
type FileReader struct{}

var _ Reader = &FileReader{}

func (r *FileReader) Read(filePaths FilePaths) ([]ast.FileObject, status.MultiError) {
	var objs []ast.FileObject
	var errs status.MultiError
	for _, f := range filePaths.Files {
		newObjs, err := r.read(filePaths.RootDir, filePaths.PolicyDir, f)
		if err != nil {
			errs = status.Append(errs, err)
			continue
		}
		objs = append(objs, newObjs...)
	}
	if errs != nil {
		return nil, errs
	}
	return objs, nil
}

// Read implements Reader.
func (r *FileReader) read(rootDir cmpath.Absolute, policyDir cmpath.Relative, file cmpath.Absolute) ([]ast.FileObject, status.MultiError) {
	unstructureds, err := parseFile(file.OSPath())
	if err != nil {
		return nil, status.PathWrapError(err, file.OSPath())
	}

	var fileObjects []ast.FileObject
	var errs status.MultiError
	for _, u := range unstructureds {
		newFileObjects, err := toFileObjects(u, rootDir, policyDir, file)
		if err != nil {
			errs = status.Append(errs, err)
		}
		fileObjects = append(fileObjects, newFileObjects...)
	}

	return fileObjects, errs
}

// toFileObjects returns either:
// 1) a slice containing a single FileObject, if the passed Unstructured is a valid Kubernetes object;
// 2) a list of multiple FileObject, if the passed Unstructured was a List type; or
// 3) an error, if there was a problem parsing the Unstructured.
func toFileObjects(u runtime.Unstructured, rootDir cmpath.Absolute, policyDir cmpath.Relative, path cmpath.Absolute) ([]ast.FileObject, status.MultiError) {
	if isList(u) {
		return flattenList(u, rootDir, policyDir, path)
	}

	oid, errs := parseID(u.UnstructuredContent(), path)
	if errs != nil {
		return nil, status.ResourceErrorBuilder.Sprint(strings.Join(errs, "\n")).
			BuildWithResources(oid)
	}

	isNomosObject := u.GetObjectKind().GroupVersionKind().Group == configmanagement.GroupName
	if !isNomosObject && hasStatusField(u) {
		return nil, syntax.IllegalFieldsInConfigError(oid, id.Status)
	}

	// This is a workaround for k8s.io/apimachinery. Such invalid fields result in
	// apply errors returned by the API Server. Rather than rely on that behavior,
	// we return an error here. This is better UX as it allows this set of very
	// common user errors to be caught by nomos vet or in the Parser, rather than
	// in the sycing step. As-is the user can catch this before they've committed
	// the mistake to Git.
	if err := validateMetadata(u, oid); err != nil {
		return nil, err
	}

	obj, ok := u.(core.Object)
	if !ok {
		// The type doesn't declare required fields, but is registered.
		// User-specified types are implicitly Unstructured, which defines Labels/Annotations/etc. even
		// if the underlying type definition does _NOT_. It isn't clear how this code would ever be reached.
		return nil, status.InternalErrorf("not a valid persistent Kubernetes type: %s", obj.GroupVersionKind().String())
	}

	// Unmarshalling and re-marshalling an object can result in spurious JSON fields depending on what
	// directives are specified for those fields. To be safe, we keep all resources in their raw
	// unstructured format unless we specifically require them for importer pre-processing. These
	// resources are mostly limited to ACM custom resources which we know are safe.
	if mustBeStructured(obj.GroupVersionKind()) {
		rObj, err := asDefaultVersioned(obj)
		if err != nil {
			return nil, ObjectParseError(oid, err)
		}
		var ok bool
		if obj, ok = rObj.(core.Object); !ok {
			return nil, ObjectParseError(oid, errNotKubernetes)
		}
	}

	rel, err := filepath.Rel(rootDir.OSPath(), path.OSPath())
	if err != nil {
		return nil, status.UndocumentedErrorBuilder.Sprintf("unable to get relative path to %s", rootDir.OSPath()).
			BuildWithPaths(path)
	}

	return []ast.FileObject{ast.NewFileObject(obj, cmpath.RelativeOS(rel))}, nil
}

func flattenList(list runtime.Unstructured, rootDir cmpath.Absolute, policyDir cmpath.Relative, path cmpath.Absolute) ([]ast.FileObject, status.MultiError) {
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
		newObjs, newErrs := toFileObjects(unstructuredItem, rootDir, policyDir, path)
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

var errNotKubernetes = errors.New("converted Kubernetes object to non-Kubernetes type")

// asDefaultVersioned converts a runtime.Object to the literal Go struct, if
// one is available. Returns an error if this process fails.
func asDefaultVersioned(obj runtime.Object) (runtime.Object, error) {
	// Determine the GroupVersion to convert the object to.
	groupVersioner := runtime.GroupVersioner(schema.GroupVersions(scheme.Scheme.PrioritizedVersionsAllGroups()))
	if _, ok := groupVersioner.KindForGroupVersionKinds([]schema.GroupVersionKind{obj.GetObjectKind().GroupVersionKind()}); !ok {
		// If the scheme doesn't have the GVK, try to serialize to the declared GV.
		groupVersioner = obj.GetObjectKind().GroupVersionKind().GroupVersion()
	}

	converter := runtime.ObjectConvertor(scheme.Scheme)
	return converter.ConvertToVersion(obj, groupVersioner)
}

// asDefaultVersionedOrOriginal returns the object as a Go object in the external form.
// If the GVK is registered in scheme.Scheme, return that version. Otherwise, try to return the declared version.
// If this fails, returns the original runtime.Unstructured.
func asDefaultVersionedOrOriginal(obj runtime.Object) runtime.Object {
	if cObj, err := asDefaultVersioned(obj); err == nil {
		return cObj
	}
	return obj
}

// validateMetadata returns a status.MultiError if metadata.annotations/labels
// has a value that wasn't parsed as a string.
func validateMetadata(u runtime.Unstructured, oid core.IDPath) status.ResourceError {
	content := u.UnstructuredContent()

	metadata, hasMetadata := content["metadata"].(map[string]interface{})
	if !hasMetadata {
		return status.ResourceErrorBuilder.Sprint("resource does not define metadata").BuildWithResources(oid)
	}

	// We could try to json.Unmarshal into an metav1.ObjectMeta, which would catch
	// these, but the returned error is very generic and unhelpful as it does not
	// help finding which value is invalid - problematic for objects with many
	// annotations or labels, or which have an unmarshalling error elsewhere.
	// As-is does miss structural errors elsewhere in metadata, but that is out of
	// scope of this code and such errors are far less common than
	// labels/annotations.
	if annotations, hasAnnotations := metadata["annotations"]; hasAnnotations {
		invalidAnnotations, err := getInvalidKeys(annotations)
		if err != nil {
			err = errors.Wrap(err, "validating annotations")
			return status.ResourceErrorBuilder.Wrap(err).BuildWithResources(oid)
		}
		if len(invalidAnnotations) > 0 {
			return InvalidAnnotationValueError(oid, invalidAnnotations)
		}
	}

	if labels, hasLabels := metadata["labels"]; hasLabels {
		invalidLabels, err := getInvalidKeys(labels)
		if err != nil {
			err = errors.Wrap(err, "validating labels")
			return status.ResourceErrorBuilder.Wrap(err).BuildWithResources(oid)
		}
		if len(invalidLabels) > 0 {
			return InvalidAnnotationValueError(oid, invalidLabels)
		}
	}

	return nil
}

var errNotAMap = errors.New("not a map")

func getInvalidKeys(o interface{}) ([]string, error) {
	if o == nil {
		return nil, nil
	}
	m, isMap := o.(map[string]interface{})
	if !isMap {
		// We don't expect this error to be thrown since the parser before it would
		// already return an error. Thus, creating a type just for this case would
		// be overkill.
		return nil, fmt.Errorf("%w: %v", errNotAMap, o)
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

// parseID makes a best-effort approach to collect information about the passed
// object.
//
// The problem is that incomplete information is useless to the user, but we
// don't know ahead of time what errors there are. Thus, we try to collect
// as much valid information as possible before exiting.
//
// Returns the best guess of the object's core.ID, and an array of encountered
// errors.
func parseID(content map[string]interface{}, path id.Path) (core.IDPath, []string) {
	oid := core.IDPath{
		Path: path,
	}
	var errs []string

	apiVersion, err := parseString(content, "apiVersion")
	if err == nil {
		gv, err := schema.ParseGroupVersion(apiVersion)
		if err == nil {
			oid.Group = gv.Group
		} else {
			errs = append(errs, err.Error())
		}
	} else {
		errs = append(errs, err.Error())
	}
	kind, err := parseString(content, "kind")
	if err == nil {
		oid.Kind = kind
	} else {
		errs = append(errs, err.Error())
	}
	name, err := parseString(content, "metadata", "name")
	if err == nil {
		oid.Name = name
	} else {
		errs = append(errs, err.Error())
	}

	return oid, errs
}

func parseString(content map[string]interface{}, fields ...string) (string, error) {
	value, hasField, e := unstructured.NestedString(content, fields...)
	if e != nil {
		return "", e
	}
	if !hasField || value == "" {
		return "", errors.Errorf("missing field %q", strings.Join(fields, "."))
	}
	return value, nil
}
