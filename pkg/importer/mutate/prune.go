package mutate

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/object"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Prune recursively removes empty maps and arrays from unstructured.Unstructured.
func Prune() object.Mutator {
	return func(object *ast.FileObject) {
		switch o := object.Object.(type) {
		case *unstructured.Unstructured:
			pruneMap(o.Object)
		default:
			panic(vet.InternalErrorf("unrecognized type %T", o))
		}
	}
}

func pruneMap(m map[string]interface{}) {
	var toRemove []string
	for k, v := range m {
		switch obj := v.(type) {
		case nil:
			toRemove = append(toRemove, k)
		case []interface{}:
			obj = pruneArray(obj)
			if len(obj) == 0 {
				toRemove = append(toRemove, k)
			}
			m[k] = obj
		case map[string]interface{}:
			pruneMap(obj)
			if len(obj) == 0 {
				toRemove = append(toRemove, k)
			}
		}
	}

	for _, key := range toRemove {
		delete(m, key)
	}
}

func pruneArray(arr []interface{}) []interface{} {
	var result []interface{}
	for _, o := range arr {
		switch obj := o.(type) {
		case nil:
			// Don't add nil entries.
		case []interface{}:
			obj = pruneArray(obj)
			if len(obj) > 0 {
				result = append(result, obj)
			}
		case map[string]interface{}:
			pruneMap(obj)
			if len(obj) > 0 {
				result = append(result, obj)
			}
		default:
			result = append(result, obj)
		}
	}
	return result
}
