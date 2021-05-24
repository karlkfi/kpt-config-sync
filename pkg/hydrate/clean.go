package hydrate

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/validate/raw/validate"
)

// Clean removes invalid fields from objects before writing them to a file.
func Clean(objects []ast.FileObject) {
	for _, o := range objects {
		clean(o)
	}
}

func clean(o ast.FileObject) {
	annotations := o.GetAnnotations()
	for k := range annotations {
		if validate.IsInvalidAnnotation(k) {
			delete(annotations, k)
		}
	}
	o.SetAnnotations(annotations)

	labels := o.GetLabels()
	for k := range labels {
		if validate.IsInvalidLabel(k) {
			delete(labels, k)
		}
	}
	o.SetLabels(labels)
}
