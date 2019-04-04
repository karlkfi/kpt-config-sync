package mutate

import (
	"strings"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/object"
)

// RemoveAnnotation removes the annotation matching annotation if it exists.
func RemoveAnnotation(annotation string) object.Mutator {
	return func(obj *ast.FileObject) {
		object.RemoveAnnotations(obj.MetaObject(), annotation)
	}
}

// RemoveAnnotationGroup removes all annotations in group.
func RemoveAnnotationGroup(group string) object.Mutator {
	return func(obj *ast.FileObject) {
		annotations := obj.MetaObject().GetAnnotations()
		var toRemove []string
		for annotation := range annotations {
			if strings.HasPrefix(annotation, group+"/") {
				toRemove = append(toRemove, annotation)
			}
		}
		object.RemoveAnnotations(obj.MetaObject(), toRemove...)
	}
}

// RemoveLabel removes the label matching label if it exists.
func RemoveLabel(label string) object.Mutator {
	return func(obj *ast.FileObject) {
		object.RemoveLabels(obj.MetaObject(), label)
	}
}

// RemoveLabelGroup removes all labels in group.
func RemoveLabelGroup(group string) object.Mutator {
	return func(obj *ast.FileObject) {
		labels := obj.MetaObject().GetLabels()
		var toRemove []string
		for label := range labels {
			if strings.HasPrefix(label, group+"/") {
				toRemove = append(toRemove, label)
			}
		}
		object.RemoveLabels(obj.MetaObject(), toRemove...)
	}
}
