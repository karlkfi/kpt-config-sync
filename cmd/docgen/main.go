package main

import (
	"os"
	"path/filepath"

	"github.com/google/nomos/pkg/policyimporter/analyzer/validation"
	"github.com/pkg/errors"
)

const (
	// Marks that the file is autogenerated.
	// This is the only platform-independent way of creating a comment in Markdown.
	autogenString = `[//]: # (Autogenerated file. Do not manually modify.)
`

	errorPreambleTmplString = `
# KNV{{.Code}}: {{.Aka}}
`

	errorExampleTmplString = `
Sample Error Message:

{{.Sample}}
`
)

var (
	docsPath = filepath.Join("docs", "user", "errors")
)

// Automatically generate documentation

// Main generates error documentation
func main() {
	if err := os.RemoveAll(filepath.Join(docsPath, "*")); err != nil {
		panic(errors.Wrap(err, "unable to clear old docs"))
	}

	if err := writeReadme(); err != nil {
		panic(errors.Wrap(err, "error writing README.md"))
	}

	for _, code := range codes {
		if err := code.document(); err != nil {
			panic(errors.Wrapf(err, "error writing documentation for %s", code))
		}
	}
}

// Add documented errors here. Adding errors which do not have validation.Example() or
// validation.Explanation() defined for them will cause a panic().
var codes = []errorDocCode{
	validation.ReservedDirectoryNameErrorCode,
	validation.InvalidNamespaceNameErrorCode,
}
