package util

import (
	"fmt"
	"os"
)

// PrintErr attempts to print an error to STDERR.
func PrintErr(err error) error {
	_, printErr := fmt.Fprintln(os.Stderr, err)
	return printErr
}

// PrintErrOrDie attempts to print an error to STDERR, and panics if it is unable to.
func PrintErrOrDie(err error) {
	printErr := PrintErr(err)
	if printErr != nil {
		panic(printErr)
	}
}
