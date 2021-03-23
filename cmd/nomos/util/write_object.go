package util

import (
	"os"
	"path/filepath"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"k8s.io/cli-runtime/pkg/printers"
)

// WriteObject writes a FileObject to a file using the provided ResourcePrinter.
// Writes to the file at object.OSPath(), overwriting if one exists.
func WriteObject(printer printers.ResourcePrinter, dir string, object ast.FileObject) error {
	if err := os.MkdirAll(filepath.Join(dir, object.Dir().OSPath()), 0750); err != nil {
		return err
	}

	file, err := os.Create(filepath.Join(dir, object.OSPath()))
	if err != nil {
		return err
	}

	return printer.PrintObj(object.Unstructured, file)
}
