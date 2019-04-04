package object

// Annotated is anything that has mutable annotations.  This is a subset of
// the interface metav1.Object, and allows us to manipulate AST objects with
// the same code that operates on Kubernetes API objects, without the need to
// implement parts of metav1.Object that don't deal with annotations.
type Annotated interface {
	GetAnnotations() map[string]string
	SetAnnotations(map[string]string)
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

// RemoveAnnotations removes the passed set of annotations from obj.
func RemoveAnnotations(obj Annotated, annotations ...string) {
	as := obj.GetAnnotations()
	for _, a := range annotations {
		delete(as, a)
	}
	obj.SetAnnotations(as)
}

// SetAnnotations replaces the current set of annotations in obj with the passed ones.
func SetAnnotations(obj Annotated, as map[string]string) {
	obj.SetAnnotations(copyMap(as))
}

// Labeled is anything containing a map of labels that can be retrieved and set.
// It is a subset of metav1.Object.
type Labeled interface {
	GetLabels() map[string]string
	SetLabels(map[string]string)
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

// RemoveLabels removes the passed list of labels from obj.
func RemoveLabels(obj Labeled, labels ...string) {
	ls := obj.GetLabels()
	for _, l := range labels {
		delete(ls, l)
	}
	obj.SetLabels(ls)
}

// SetLabels replaces the current set of labels in obj with the passed ones.
func SetLabels(obj Labeled, labels map[string]string) {
	obj.SetLabels(copyMap(labels))
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
