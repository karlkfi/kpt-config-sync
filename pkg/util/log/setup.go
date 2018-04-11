package log

import "flag"

// Sets up default logging parameters for nomos controllers
func Setup() {
	if err := flag.Set("logtostderr", "true"); err != nil {
		panic(err)
	}
	if err := flag.Set("v", "2"); err != nil {
		panic(err)
	}
}
