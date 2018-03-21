package log

import "flag"

// Sets up default logging parameters for nomos controllers
func Setup() {
	flag.Set("logtostderr", "true")
	flag.Set("v", "2")
}
