package core

// SetAnnotation sets the annotation on the passed annotated object to value.
func SetAnnotation(obj LabeledAndAnnotated, annotation, value string) {
	as := obj.GetAnnotations()
	if as == nil {
		as = make(map[string]string)
	}
	as[annotation] = value
	obj.SetAnnotations(as)
}

// GetAnnotation gets the annotation value on the passed annotated object for a given key.
func GetAnnotation(obj LabeledAndAnnotated, annotation string) string {
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

// RemoveAnnotations removes the passed set of annotations from obj.
func RemoveAnnotations(obj LabeledAndAnnotated, annotations ...string) {
	as := obj.GetAnnotations()
	for _, a := range annotations {
		delete(as, a)
	}
	obj.SetAnnotations(as)
}

// SetLabel sets label on obj to value.
func SetLabel(obj LabeledAndAnnotated, label, value string) {
	ls := obj.GetLabels()
	if ls == nil {
		ls = make(map[string]string)
	}
	ls[label] = value
	obj.SetLabels(ls)
}

// RemoveLabels removes labels from the obj that key/value match the passed in map
func RemoveLabels(obj LabeledAndAnnotated, labels map[string]string) {
	ls := obj.GetLabels()
	for key, val := range labels {
		if _, ok := ls[key]; !ok {
			continue
		}

		if ls[key] == val {
			delete(ls, key)
		}
	}
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
