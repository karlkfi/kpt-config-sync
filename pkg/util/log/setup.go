package log

import "flag"

// Sets up default logging parameters for stolos controllers
func Setup() {
	flag.Set("logtostderr", "true")
	flag.Set("v", "2")
}