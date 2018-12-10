package main

import (
	"path/filepath"
	"text/template"
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

	if executeErr := tmpl.Execute(file, codes); executeErr != nil {
		return executeErr
	}

	return file.Close()
}
