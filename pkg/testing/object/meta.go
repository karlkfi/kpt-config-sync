package object

import "github.com/google/nomos/pkg/policyimporter/analyzer/ast"

// Namespace replaces the metadata.namesapce of the FileObject under test.
func Namespace(namespace string) BuildOpt {
	return func(o *ast.FileObject) {
		o.MetaObject().SetNamespace(namespace)
	}
}

// Name replaces the metadata.name of the FileObject under test.
func Name(name string) BuildOpt {
	return func(o *ast.FileObject) {
		o.MetaObject().SetName(name)
	}
}

// Label adds label=value to the metadata.labels of the FileObject under test.
func Label(label, value string) BuildOpt {
	return func(o *ast.FileObject) {
		labels := o.MetaObject().GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		labels[label] = value
		o.MetaObject().SetLabels(labels)
	}
}

// Labels sets the object's labels to a copy of the passed map.
// Setting to nil causes a call to SetLabels(nil), but the underlying implementation may set Labels
// to empty map.
func Labels(labels map[string]string) BuildOpt {
	return func(o *ast.FileObject) {
		o.MetaObject().SetLabels(copyMap(labels))
	}
}

// Annotation adds annotation=value to the metadata.annotations of the FileObject under test.
func Annotation(annotation, value string) BuildOpt {
	return func(o *ast.FileObject) {
		annotations := o.MetaObject().GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations[annotation] = value
		o.MetaObject().SetAnnotations(annotations)
	}
}

// Annotations sets the object's annotations to a copy of the passed map.
// Setting to nil causes a call to SetAnnotations(nil), but the underlying implementation may set
// Annotations to empty map.
func Annotations(annotations map[string]string) BuildOpt {
	return func(o *ast.FileObject) {
		o.MetaObject().SetAnnotations(copyMap(annotations))
	}
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
