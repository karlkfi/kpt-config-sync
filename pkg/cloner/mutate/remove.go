package mutate

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// KeyValue represents a YAML path, and an optional value to remove at that path.
type KeyValue struct {
	// key is the YAML path to the part of the tree to remove.
	key []string
	// value is the optional value to remove from the path. If unset, the path will be removed
	// without further checks.
	value *string
}

// Key creates a KeyValue pointing to a location in the YAML tree.
func Key(key string, keys ...string) KeyValue {
	return KeyValue{key: append([]string{key}, keys...)}
}

// Value specifies that the part of the YAML tree should only be removed if it matches value.
func (kv KeyValue) Value(value string) KeyValue {
	kv.value = &value
	return kv
}

func (kv KeyValue) len() int {
	return len(kv.key)
}

func (kv KeyValue) pop() KeyValue {
	return KeyValue{kv.key[1:], kv.value}
}

func (kv KeyValue) next() string {
	return kv.key[0]
}

// Remove returns a Mutator which removes all YAML paths matching key in an unstructured.Unstructured.
// key must have a length of at least 1.
func Remove(key KeyValue) Mutator {
	return func(object *ast.FileObject) {
		switch o := object.Object.(type) {
		case *unstructured.Unstructured:
			removeFromMap(key, o.Object)
		default:
			panic(vet.InternalErrorf("unrecognized type %T", o))
		}
	}
}

func removeFromMap(kv KeyValue, o map[string]interface{}) {
	if kv.len() == 1 && kv.value == nil {
		// Delete the entry matching key.
		delete(o, kv.next())
		return
	}

	switch obj := o[kv.next()].(type) {
	case string:
		if kv.len() == 1 {
			if obj == *kv.value {
				// The key matched this entry in the map.
				delete(o, kv.next())
			}
		}
	case []interface{}:
		// Recurse into the array.
		o[kv.next()] = removeFromArray(kv.pop(), obj)
	case map[string]interface{}:
		// Recurse into the map.
		removeFromMap(kv.pop(), obj)
	}
}

func removeFromArray(kv KeyValue, o []interface{}) []interface{} {
	var result []interface{}
	for _, i := range o {
		switch obj := i.(type) {
		case string:
			if kv.len() == 0 && (kv.value != nil) && (*kv.value) == obj {
				// This string matches key, so don't add it to result.
				continue
			}
		case []interface{}:
			// Recurse into the array.
			// We didn't match an entry in key, so don't increment depth.
			i = removeFromArray(kv, obj)
		case map[string]interface{}:
			// Recurse into the map.
			// We didn't match an entry in key, so don't increment depth.
			removeFromMap(kv, obj)
		}
		result = append(result, i)
	}
	return result
}
