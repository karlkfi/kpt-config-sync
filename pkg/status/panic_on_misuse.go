package status

import (
	"flag"

	"k8s.io/klog/v2"
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

// reportMisuse either panics, or logs an error with klog.Errorf depending on
// whether panicOnMisuse is true.
func reportMisuse(message string) {
	if panicOnMisuse {
		// We're in debug mode, so halt execution so the runner is likely to get
		// a signal that something is wrong.
		panic(message)
	} else {
		// Show it in the logs, but don't kill the application in production.
		klog.Errorf("internal error: %s", message)
	}
}
