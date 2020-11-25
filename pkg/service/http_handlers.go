// HTTP handler functions, ready for reuse.

package service

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var metricsPort = flag.Int("metrics-port", 8675, "The port to export prometheus metrics on.")

// NoCache positively turns off page caching.
func noCache(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set(
			"Cache-Control",
			"no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		handler.ServeHTTP(w, req)
	}
}

// ServePrometheusMetrics spins up a standalone metrics HTTP endpoint.
func ServePrometheusMetrics(openCensus bool) {
	var handler http.Handler
	if openCensus {
		pe, err := metrics.RegisterPrometheusExporter()
		if err != nil {
			glog.Fatalf("Failed to register Prometheus exporter: %v", err)
		}
		handler = pe
	} else {
		handler = promhttp.Handler()
	}
	// Expose prometheus metrics via HTTP.
	http.Handle("/metrics", handler)
	http.Handle("/threads", noCache(http.HandlerFunc(goRoutineHandler)))
	glog.Infof("Serving metrics on :%d/metrics", *metricsPort)
	err := http.ListenAndServe(fmt.Sprintf(":%d", *metricsPort), nil)
	if err != nil {
		glog.Fatalf("HTTP ListenAndServe for metrics: %+v", err)
	}
}
