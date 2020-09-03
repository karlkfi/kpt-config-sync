package util

import (
	"fmt"
	"os"
)

// PrintErrOrDie attempts to print an error to STDERR, and panics if it is unable to.
func PrintErrOrDie(err error) {
	_, printErr := fmt.Fprintln(os.Stderr, err)
	if printErr != nil {
		panic(printErr)
	}
}
