package util

import (
	"fmt"
	"os"

	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
)

// PrintErrAndDie prints an error to STDERR and exits immediately
func PrintErrAndDie(err error) {
	// nolint: errcheck
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

// GetRootOrDie returns a cmpath.Root of the path, or prints an error and dies if it is unable to.
func GetRootOrDie(rootDir string) cmpath.Root {
	rootPath, err := cmpath.NewRoot(cmpath.FromOS(rootDir))
	if err != nil {
		PrintErrAndDie(err)
	}
	return rootPath
}
