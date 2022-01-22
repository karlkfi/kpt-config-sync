package log

import (
	"flag"

	"github.com/google/nomos/pkg/version"
	"k8s.io/klog/v2"
)

// Setup sets up default logging configs for Nomos applications and logs the preamble.
func Setup() {
	klog.InitFlags(nil)
	if err := flag.Set("logtostderr", "true"); err != nil {
		klog.Fatal(err)
	}
	flag.Parse()
	klog.Infof("Build Version: %s", version.VERSION)
}
