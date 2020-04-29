package filesystem

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang/glog"
)

// WalkDirectory walks a directory and returns a list of all dirs / files / errors.
func WalkDirectory(dir string) ([]string, error) {
	var seen []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			seen = append(seen, fmt.Sprintf("path=%s error=%s", path, err))
			return nil
		}
		seen = append(seen, fmt.Sprintf("path=%s mode=%o size=%d mtime=%s", path, info.Mode(), info.Size(), info.ModTime()))
		return nil
	})
	return seen, err
}

// LogWalkDirectory logs a directory walk to glog.Error
func LogWalkDirectory(dir string) {
	files, err := WalkDirectory(dir)
	if err != nil {
		glog.Errorf("error while walking %s: %s", dir, err)
	}
	glog.Errorf("walked %d files in %s", len(files), dir)
	for _, file := range files {
		glog.Errorf("  %s", file)
	}
}
