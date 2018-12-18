package util

import (
	"fmt"
	"os"
)

// PrintErrAndDie prints an error to STDERR and exits immediately
func PrintErrAndDie(err error) {
	// nolint: errcheck
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
