package mutate

import (
	"strings"

	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
)

// RemoveAnnotation removes the annotation matching annotation if it exists.
func RemoveAnnotation(annotation string) Mutator {
	return func(object *ast.FileObject) {
		annotations := object.MetaObject().GetAnnotations()
		delete(annotations, annotation)
		// Required since GetAnnotations implementations are inconsistent and some return copies.
		object.MetaObject().SetAnnotations(annotations)
	}
}

// RemoveAnnotationGroup removes all annotations in group.
func RemoveAnnotationGroup(group string) Mutator {
	return func(object *ast.FileObject) {
		annotations := object.MetaObject().GetAnnotations()
		var toRemove []string
		for annotation := range annotations {
			if strings.HasPrefix(annotation, group+"/") {
				toRemove = append(toRemove, annotation)
			}
		}
		for _, annotation := range toRemove {
			delete(annotations, annotation)
		}
		// Required since GetAnnotations implementations are inconsistent and some return copies.
		object.MetaObject().SetAnnotations(annotations)
	}
}

// RemoveLabel removes the label matching label if it exists.
func RemoveLabel(label string) Mutator {
	return func(object *ast.FileObject) {
		labels := object.MetaObject().GetLabels()
		delete(labels, label)
		// Required since GetLabels implementations are inconsistent and some return copies.
		object.MetaObject().SetLabels(labels)
	}
}

// RemoveLabelGroup removes all labels in group.
func RemoveLabelGroup(group string) Mutator {
	return func(object *ast.FileObject) {
		labels := object.MetaObject().GetLabels()
		var toRemove []string
		for label := range labels {
			if strings.HasPrefix(label, group+"/") {
				toRemove = append(toRemove, label)
			}
		}
		for _, l := range toRemove {
			delete(labels, l)
		}
		// Required since GetLabels implementations are inconsistent and some return copies.
		object.MetaObject().SetLabels(labels)
	}
}
