package core

import "sigs.k8s.io/controller-runtime/pkg/client"

// Annotated is the interface defined by types with annotations. Note that
// some non-objects (such as PodTemplates) define annotations but are not objects.
type Annotated interface {
	GetAnnotations() map[string]string
	SetAnnotations(annotations map[string]string)
}

// SetAnnotation sets the annotation on the passed annotated object to value.
func SetAnnotation(obj Annotated, annotation, value string) {
	as := obj.GetAnnotations()
	if as == nil {
		as = make(map[string]string)
	}
	as[annotation] = value
	obj.SetAnnotations(as)
}

// GetAnnotation gets the annotation value on the passed annotated object for a given key.
func GetAnnotation(obj client.Object, annotation string) string {
	as := obj.GetAnnotations()
	if as == nil {
		return ""
	}
	value, found := as[annotation]
	if found {
		return value
	}
	return ""
}

// GetLabel gets the label value on the passed object for a given key.
func GetLabel(obj client.Object, label string) string {
	as := obj.GetLabels()
	if as == nil {
		return ""
	}
	value, found := as[label]
	if found {
		return value
	}
	return ""
}

// RemoveAnnotations removes the passed set of annotations from obj.
func RemoveAnnotations(obj client.Object, annotations ...string) {
	as := obj.GetAnnotations()
	for _, a := range annotations {
		delete(as, a)
	}
	obj.SetAnnotations(as)
}

// Labeled is the interface defined by types with labeled. Note that
// some non-objects (such as PodTemplates) define labels but are not objects.
type Labeled interface {
	GetLabels() map[string]string
	SetLabels(annotations map[string]string)
}

// SetLabel sets label on obj to value.
func SetLabel(obj Labeled, label, value string) {
	ls := obj.GetLabels()
	if ls == nil {
		ls = make(map[string]string)
	}
	ls[label] = value
	obj.SetLabels(ls)
}

// copyMap returns a copy of the passed map. Otherwise the Labels or Annotations maps will have two
// owners.
func copyMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	result := make(map[string]string)
	for k, v := range m {
		result[k] = v
	}
	return result
}
