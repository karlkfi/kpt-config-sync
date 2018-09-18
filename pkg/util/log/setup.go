package log

import (
	"flag"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/version"
)

// Setup sets up default logging configs for Nomos applications and logs the preamble.
func Setup() {
	if err := flag.Set("logtostderr", "true"); err != nil {
		glog.Fatal(err)
	}
	glog.Infof("Build Version: %s", version.VERSION)
}
