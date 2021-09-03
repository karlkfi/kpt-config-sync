package profiler

import (
	"flag"
	"fmt"
	"net/http"

	// Empty import as required by pprof
	_ "net/http/pprof"

	"github.com/golang/glog"
)

var enableProfiler = flag.Bool("enable-pprof", false, "enable pprof profiling")
var profilerPort = flag.Int("pprof-port", 6060, "port for pprof profiling. defaulted to 6060 if unspecified")

// Service starts the profiler http endpoint if --enable-pprof flag is passed
func Service() {
	if *enableProfiler {
		go func() {
			glog.Infof("Starting profiling on port %d", *profilerPort)
			addr := fmt.Sprintf(":%d", *profilerPort)
			err := http.ListenAndServe(addr, nil)
			if err != nil {
				glog.Fatalf("Profiler server failed to start: %+v", err)
			}
		}()
	}
}
