package log

import "flag"

// Setup sets up default logging parameters for nomos controllers
func Setup() {
	if err := flag.Set("logtostderr", "true"); err != nil {
		panic(err)
	}
	if err := flag.Set("v", "2"); err != nil {
		panic(err)
	}
}
