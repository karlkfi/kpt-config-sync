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
