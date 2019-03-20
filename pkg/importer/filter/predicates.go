package filter

import (
	"strings"

	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Group returns true for resources matching the passed Group.
func Group(group string) object.Predicate {
	return func(object ast.FileObject) bool {
		return object.GroupVersionKind().Group == group
	}
}

// Kind returns true for resources matching the passed Kind.
func Kind(kind string) object.Predicate {
	return func(object ast.FileObject) bool {
		return object.GroupVersionKind().Kind == kind
	}
}

// GroupKind returns true for resources matching the passed GroupKind.
func GroupKind(groupKind schema.GroupKind) object.Predicate {
	return object.All(Group(groupKind.Group), Kind(groupKind.Kind))
}

// Namespace returns true if the object has metadata.Namespace set to the provided value or is the
// Namespace.
func Namespace(namespace string) object.Predicate {
	hasNamespace := object.Predicate(func(object ast.FileObject) bool {
		return namespace == object.MetaObject().GetNamespace()
	})
	isNamespace := object.All(GroupKind(kinds.Namespace().GroupKind()), Name(namespace))
	return object.Any(hasNamespace, isNamespace)
}

// Name returns true if the object's metadata.name exactly matches name.
func Name(name string) object.Predicate {
	return func(object ast.FileObject) bool {
		return object.Name() == name
	}
}

// NameGroup returns true if the object's metadata.name is in the passed name group.
func NameGroup(nameGroup string) object.Predicate {
	return func(object ast.FileObject) bool {
		return strings.HasPrefix(object.Name(), nameGroup+":")
	}
}

// Label returns true if the object has a metadata.label exactly matching label.
func Label(label string) object.Predicate {
	return func(object ast.FileObject) bool {
		_, found := object.MetaObject().GetLabels()[label]
		return found
	}
}
