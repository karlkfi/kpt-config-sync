package util

import "github.com/google/nomos/pkg/importer/filesystem/cmpath"

// GetRootOrDie returns a cmpath.Root of the path, or prints an error and dies if it is unable to.
func GetRootOrDie(rootDir string) cmpath.Root {
	rootPath, err := cmpath.NewRoot(cmpath.FromOS(rootDir))
	if err != nil {
		PrintErrAndDie(err)
	}
	return rootPath
}
