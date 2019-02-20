package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/policyhierarchy"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/node"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/pkg/errors"
)

// For use in templating.
type errorDocCode string

// Fully document an error type.
func (e errorDocCode) document(docsPath string) error {
	glog.Error(e.errorFile(docsPath))
	file := openOrDie(e.errorFile(docsPath))

	if _, writeErr := file.WriteString(autogenString); writeErr != nil {
		return errors.Wrap(writeErr, "programmer error: unable to write autogen string")
	}

	if preambleErr := e.writePreamble(file); preambleErr != nil {
		return errors.Wrap(preambleErr, "programmer error: unable to write preamble")
	}

	if explanationErr := e.writeExplanation(file); explanationErr != nil {
		return errors.Wrap(explanationErr, "programmer error: unable to write explanation")
	}

	return file.Close()
}

func (e errorDocCode) writePreamble(wr io.Writer) error {
	return e.execute(wr, errorPreambleTmplString, "Preamble")
}

func (e errorDocCode) writeExplanation(wr io.Writer) error {
	return e.execute(wr, vet.Explanations[e.Code()], "Explanation")
}

// CodeMode enters and exits multiline monospace mode.
func (e errorDocCode) CodeMode() string {
	return "```"
}

// execute is a convenience method for templating logic.
func (e errorDocCode) execute(wr io.Writer, templateStr string, name string) error {
	tmpl, parseErr := template.New(fmt.Sprintf("%s %s", name, e.Code())).Parse(templateStr)
	if parseErr != nil {
		return parseErr
	}
	return tmpl.Execute(wr, e)
}

// openOrDie is a convenience method that dies if a file is unopenable.
func openOrDie(path string) *os.File {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		fmt.Println(errors.Wrapf(err, "programmer error: unable to open file %q", path))
		os.Exit(1)
	}
	return file
}

func (e errorDocCode) errorFile(docsPath string) string {
	return filepath.Join(docsPath, e.ErrorFileBase())
}

// ErrorFileBase returns the base file of the error doc.
func (e errorDocCode) ErrorFileBase() string {
	return fmt.Sprintf("knv%s.md", e.Code())
}

// The below methods aren't really meant to be exported, but are here because Templates require
// the methods to be export-able.

// Code returns the error code
func (e errorDocCode) Code() string {
	return string(e)
}

// Examples returns the example errors
func (e errorDocCode) Examples() []vet.Error {
	return vet.Examples[e.Code()]
}

// Aka returns the type of error in a near-human-readable format
func (e errorDocCode) Aka() string {
	return strings.Split(fmt.Sprintf("%T", e.Examples()[0]), "vet.")[1]
}

// Nomos returns `nomos`
func (e errorDocCode) Nomos() string {
	return e.Q(policyhierarchy.CLIName)
}

// Nomosvet returns `nomos vet`
func (e errorDocCode) Nomosvet() string {
	return e.Q(fmt.Sprintf("%s vet", policyhierarchy.CLIName))
}

// Namespace returns the Namespace object string
func (e errorDocCode) Namespace() string {
	return string(node.Namespace)
}

// NamespacesDir returns the dir holding Namespaces
func (e errorDocCode) NamespacesDir() string {
	return "`" + repo.NamespacesDir + "/`"
}

// Q puts the passed parameter in monospace mode.
// The method name is short because it is going to appear everywhere. It will harm the readability
// of the template code for this to be anything longer.
func (e errorDocCode) Q(s string) string {
	return "`" + s + "`"
}
