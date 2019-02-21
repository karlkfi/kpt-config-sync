package service

import (
	"fmt"
	"net/http"
	"runtime/pprof"
)

// GoRoutineHandler is a handler that will print the goroutine stacks to the response.
func GoRoutineHandler(w http.ResponseWriter, _ *http.Request) {
	ps := pprof.Profiles()
	for _, p := range ps {
		if p.Name() == "goroutine" {
			if err := p.WriteTo(w, 2); err != nil {
				response := fmt.Sprintf("error while writing goroutine stakcs: %s", err)
				// nolint:errcheck
				w.Write([]byte(response))
			}
			return
		}
	}

	response := "unable to find profile for goroutines"
	// nolint:errcheck
	w.Write([]byte(response))
}
