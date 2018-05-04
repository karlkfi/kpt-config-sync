package log

import (
	"flag"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/version"
)

// Setup sets up default logging configs for Nomos applications and logs the preamble.
func Setup() {
	if err := flag.Set("logtostderr", "true"); err != nil {
		panic(err)
	}
	if err := flag.Set("v", "2"); err != nil {
		panic(err)
	}
	glog.Infof("Build Version: %s", version.VERSION)
}
