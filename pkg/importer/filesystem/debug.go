package filesystem

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/klog/v2"
)

// WalkDirectory exported for testing.
var WalkDirectory = walkDirectory

// walkDirectory walks a directory and returns a list of all dirs / files / errors.
func walkDirectory(dir string) ([]string, error) {
	var seen []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			seen = append(seen, fmt.Sprintf("path=%s error=%s", path, err))
			return nil
		}
		seen = append(seen, fmt.Sprintf("path=%s mode=%o size=%d mtime=%s", path, info.Mode(), info.Size(), info.ModTime()))
		// Skip .git subdirectories.
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}
		return nil
	})
	return seen, err
}

// logWalkDirectory logs a directory walk to klog.Error
func logWalkDirectory(dir string) {
	files, err := walkDirectory(dir)
	if err != nil {
		klog.Errorf("error while walking %s: %s", dir, err)
	}
	klog.Errorf("walked %d files in %s", len(files), dir)
	for _, file := range files {
		klog.Errorf("  %s", file)
	}
}
