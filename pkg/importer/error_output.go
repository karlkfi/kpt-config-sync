package importer

import (
	"errors"
	"os"
)

// ErrorOutput provides common functions for handling errors.
type ErrorOutput struct {
	// out is the output stream to write errors to.
	out *os.File

	// err tracks whether any errors have been output.
	err bool
}

// Add prints an error to the output stream.
// Subsequent calls to DieIfPrintedErrors will terminate the program with exit code 1.
// Panics if there is an error writing to the error output stream.
// If err is nil, has no effect.
func (eo *ErrorOutput) Add(err error) {
	if err == nil {
		return
	}
	eo.err = true
	_, printErr := eo.out.WriteString(err.Error() + "\n")
	if printErr != nil {
		panic(printErr)
	}
}

// AddAndDie prints an error to the output stream and exits immediately with exit code 1.
// If err is nil, has no effect.
func (eo *ErrorOutput) AddAndDie(err error) {
	if err == nil {
		return
	}
	eo.Add(err)
	eo.DieIfPrintedErrors("Encountered fatal error")
}

// DieIfPrintedErrors exits if any errors have been printed previously.
func (eo *ErrorOutput) DieIfPrintedErrors(message string) {
	if eo.err {
		eo.Add(errors.New(message))
		os.Exit(1)
	}
}
