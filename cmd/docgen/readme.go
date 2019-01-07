package main

import (
	"path/filepath"
	"text/template"

	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors"
)

const (
	readmeTmplString = `
# Validation Errors

Errors ` + "`nomos vet`" + ` may throw while analyzing a GKE Policy Management directory.

{{ range $index, $err := . }}*   [KNV{{$err.Code}}: {{$err.Aka}}]({{.ErrorFileBase}})
{{ end }}`
)

// Path to the nomos vet errors README.md file.
var (
	readmeFile = filepath.Join(docsPath, "README.md")
)

// writeReadme Writes the README.md which offers an overview of the nomos vet errors and what these
// explanations are for. Also includes a list of errors and links to their respective pages.
func writeReadme() error {
	file := openOrDie(readmeFile)

	tmpl, parseErr := template.New("Readme").Parse(readmeTmplString)
	if parseErr != nil {
		return parseErr
	}

	if _, writeErr := file.WriteString(autogenString); writeErr != nil {
		return writeErr
	}

	errorDocCodes := make([]errorDocCode, len(veterrors.Examples))
	for code, example := range veterrors.Examples {
		if example == nil {
			continue
		}
		errorDocCodes = append(errorDocCodes, errorDocCode(code))
	}
	if executeErr := tmpl.Execute(file, errorDocCodes); executeErr != nil {
		return executeErr
	}

	return file.Close()
}
