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
	//
	// rootDir is the absolute path to policyDir.
	// files is the list of absolute path to the files to read.
	// TODO(b/155509765): These two arguments get passed around together a lot
	//  and have the same lifecycle. Encapsulate in a struct or similar.
	Read(rootDir cmpath.Absolute, files []cmpath.Absolute) ([]ast.FileObject, status.MultiError)
}

// FileReader reads FileObjects from a filesystem.
type FileReader struct{}

var _ Reader = &FileReader{}

func (r *FileReader) Read(rootDir cmpath.Absolute, files []cmpath.Absolute) ([]ast.FileObject, status.MultiError) {
	var objs []ast.FileObject
	var errs status.MultiError
	for _, f := range files {
		newObjs, err := r.read(rootDir, f)
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
func (r *FileReader) read(rootDir cmpath.Absolute, file cmpath.Absolute) ([]ast.FileObject, status.MultiError) {
	unstructureds, err := parseFile(file.OSPath())
	if err != nil {
		return nil, status.PathWrapError(err, file.OSPath())
	}

	var fileObjects []ast.FileObject
	var errs status.MultiError
	for _, u := range unstructureds {
		newFileObjects, err := toFileObjects(u, rootDir, file)
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
func toFileObjects(u runtime.Unstructured, rootDir, path cmpath.Absolute) ([]ast.FileObject, status.MultiError) {
	if isList(u) {
		return flattenList(u, rootDir, path)
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

	// TODO: Remove the validateMetadata check when we migrate to
	//  apimachinery 1.17 since it is fixed then.
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
	if syntax.MustBeStructured(obj.GroupVersionKind()) {
		rObj, err := asDefaultVersioned(obj)
		if err != nil {
			return nil, syntax.ObjectParseError(oid, err)
		}
		var ok bool
		if obj, ok = rObj.(core.Object); !ok {
			return nil, syntax.ObjectParseError(oid, errNotKubernetes)
		}
	}

	rel, err := filepath.Rel(rootDir.OSPath(), path.OSPath())
	if err != nil {
		return nil, status.UndocumentedErrorBuilder.Sprintf("unable to get relative path to %s", rootDir.OSPath()).
			BuildWithPaths(path)
	}

	return []ast.FileObject{ast.NewFileObject(obj, cmpath.RelativeOS(rel))}, nil
}

func flattenList(list runtime.Unstructured, rootDir, path cmpath.Absolute) ([]ast.FileObject, status.MultiError) {
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
		newObjs, newErrs := toFileObjects(unstructuredItem, rootDir, path)
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
// has a value that wasn't parsed as a string. This is a workaround the
// k8s.io/apimachinery bug that causes the entire annotations field to be
// silently discarded if any values aren't strings.
//
// TODO(b/154838005): Remove this once we upgrade and don't need to explicitly
//  check for this.
func validateMetadata(u runtime.Unstructured, oid core.IDPath) status.ResourceError {
	content := u.UnstructuredContent()

	metadata, hasMetadata := content["metadata"].(map[string]interface{})
	if !hasMetadata {
		return status.ResourceErrorBuilder.Sprint("resource does not define metadata").BuildWithResources(oid)
	}

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
