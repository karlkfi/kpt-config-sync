package util

import (
	"fmt"
	"os"
)

// PrintErrAndDie prints an error to STDERR and exits immediately
func PrintErrAndDie(err error) {
	PrintErrOrDie(err)
	os.Exit(1)
}

// PrintErrOrDie attempts to print an error to STDERR, and panics if it is unable to.
func PrintErrOrDie(err error) {
	_, printErr := fmt.Fprintln(os.Stderr, err)
	if printErr != nil {
		panic(printErr)
	}
}
