package filesystem

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
)

func isHierarchyFile(root cmpath.Absolute, file cmpath.Absolute) bool {
	fileSplits := file.Split()
	rootSplits := root.Split()
	if len(fileSplits) < len(rootSplits) {
		return false
	}
	for i := range rootSplits {
		if fileSplits[i] != rootSplits[i] {
			return false
		}
	}
	return fileSplits[len(rootSplits)] == repo.SystemDir ||
		fileSplits[len(rootSplits)] == repo.ClusterDir ||
		fileSplits[len(rootSplits)] == repo.ClusterRegistryDir ||
		fileSplits[len(rootSplits)] == repo.NamespacesDir
}

// FilterHierarchyFiles filters out files that aren't in a top-level directory
// we care about.
// root and files are all absolute paths.
func FilterHierarchyFiles(root cmpath.Absolute, files []cmpath.Absolute) []cmpath.Absolute {
	var result []cmpath.Absolute
	for _, file := range files {
		if isHierarchyFile(root, file) {
			result = append(result, file)
		}
	}
	return result
}
