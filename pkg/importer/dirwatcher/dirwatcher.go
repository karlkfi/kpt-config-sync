package dirwatcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang/glog"
)

type scan struct {
	start    time.Time
	end      time.Time
	manifest map[string]os.FileInfo
}

func newScan() *scan {
	return &scan{
		start:    time.Now(),
		manifest: map[string]os.FileInfo{},
	}
}

func (s *scan) setEnd() {
	if !s.end.IsZero() {
		panic("end already set")
	}
	s.end = time.Now()
}

// Watcher performs a recursive watch on files in a directory
type Watcher struct {
	dir  string
	scan *scan
}

// NewWatcher creates a new watcher for the given directory.
func NewWatcher(dir string) *Watcher {
	glog.Infof("Setting up recursive watch for %s", dir)
	w := &Watcher{
		dir:  dir,
		scan: newScan(),
	}
	w.scan.setEnd()
	return w
}

// Watch starts a watch on the directory with the given probe period and stop channel.
func (w *Watcher) Watch(ctx context.Context, period time.Duration) {
	w.probe(w.dir)
	for {
		select {
		case <-time.After(period):
			w.probe(w.dir)
		case <-ctx.Done():
			glog.Infof("Stop requested, terminating watch")
			return
		}
	}
}

func (w *Watcher) probe(dir string) {
	scan := newScan()
	if _, lstatErr := os.Lstat(dir); lstatErr == nil {
		walkErr := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				glog.Errorf("got error while walking %s at %s", dir, path)
				return err
			}
			scan.manifest[path] = info
			if prevInfo, found := w.scan.manifest[path]; found {
				if diff := w.diff(prevInfo, info); diff != "" {
					glog.Infof("modify %s %s %s", typeStr(info), path, diff)
				}
			} else {
				// add
				glog.Infof("add %s %s %s", typeStr(info), path, w.infoStr(info))
			}
			return nil
		})
		if walkErr != nil {
			glog.Errorf("got error from filepath.Walk for %s: %v", dir, walkErr)
		}
	} else {
		switch {
		case os.IsNotExist(lstatErr):
		// does not exist, diff logic will take care of printing removal
		default:
			glog.Errorf("got error from os.Lstat %s: %v", dir, lstatErr)
		}
	}
	scan.setEnd()

	// deletes
	for path, prevInfo := range w.scan.manifest {
		if _, found := scan.manifest[path]; !found {
			glog.Infof("remove %s %s", typeStr(prevInfo), path)
		}
	}
	w.scan = scan
}

func (w *Watcher) diff(prev, cur os.FileInfo) string {
	var changes []string
	if prev.IsDir() != cur.IsDir() {
		changes = append(changes, fmt.Sprintf("%s -> %s", typeStr(prev), typeStr(cur)))
	}
	if prev.Size() != cur.Size() {
		changes = append(changes, fmt.Sprintf("size %d -> %d", prev.Size(), cur.Size()))
	}
	if prev.Mode() != cur.Mode() {
		changes = append(changes, fmt.Sprintf("mode %#o -> %#o", prev.Mode(), cur.Mode()))
	}
	if prev.ModTime() != cur.ModTime() {
		changes = append(changes, fmt.Sprintf("mtime %s -> %s", prev.ModTime(), cur.ModTime()))
	}
	if len(changes) == 0 {
		return ""
	}
	return strings.Join(changes, ", ")
}

func (w *Watcher) infoStr(info os.FileInfo) string {
	return fmt.Sprintf("size=%d mode=%#o mtime=%s", info.Size(), info.Mode(), info.ModTime())
}

func typeStr(info os.FileInfo) string {
	if info.IsDir() {
		return "dir "
	}
	return "file"
}
