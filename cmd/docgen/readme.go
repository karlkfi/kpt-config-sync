package main

import (
	"path/filepath"
	"sort"
	"text/template"

	"github.com/google/nomos/pkg/api/policyhierarchy"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
)

const (
	readmeTmplString = `
# Validation Errors

Errors ` + "`" + policyhierarchy.CLIName + " vet" + "`" + ` may throw while analyzing a GKE Policy Management directory.

{{ range $index, $err := . }}*   [KNV{{$err.Code}}: {{$err.Aka}}]({{.ErrorFileBase}})
{{ end }}`
)

// Path to the nomos vet errors README.md file.
var (
	readmeFile = "readme.md"
)

// writeReadme Writes the README.md which offers an overview of the nomos vet errors and what these
// explanations are for. Also includes a list of errors and links to their respective pages.
func writeReadme(docspath string) error {
	file := openOrDie(filepath.Join(docspath, readmeFile))

	tmpl, parseErr := template.New("Readme").Parse(readmeTmplString)
	if parseErr != nil {
		return parseErr
	}

	if _, writeErr := file.WriteString(autogenString); writeErr != nil {
		return writeErr
	}

	var errorDocCodes []errorDocCode
	for code, example := range vet.Examples {
		if example == nil {
			continue
		}
		errorDocCodes = append(errorDocCodes, errorDocCode(code))
	}
	sort.Slice(errorDocCodes, func(i, j int) bool {
		return errorDocCodes[i].Code() < errorDocCodes[j].Code()
	})
	if executeErr := tmpl.Execute(file, errorDocCodes); executeErr != nil {
		return executeErr
	}

	return file.Close()
}
