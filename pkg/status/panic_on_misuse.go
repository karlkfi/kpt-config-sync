package status

import (
	"flag"

	"github.com/golang/glog"
)

var panicOnMisuse = false

func init() {
	if flag.Lookup("test.v") != nil {
		// Running with "go test"
		EnablePanicOnMisuse()
	}
}

// EnablePanicOnMisuse makes status.Error ensure errors are properly formatted,
// aren't wrapped unnecessarily, and so on. Should only be enabled in debugging
// and test settings, not in production.
func EnablePanicOnMisuse() {
	panicOnMisuse = true
}

// reportMisuse either panics, or logs an error with glog.Errorf depending on
// whether panicOnMisuse is true.
func reportMisuse(message string) {
	if panicOnMisuse {
		// We're in debug mode, so halt execution so the runner is likely to get
		// a signal that something is wrong.
		panic(message)
	} else {
		// Show it in the logs, but don't kill the application in production.
		glog.Errorf("internal error: %s", message)
	}
}
