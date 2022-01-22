// HTTP handler functions, ready for reuse.

package service

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/klog/v2"
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

// ServeMetrics spins up a standalone metrics HTTP endpoint.
func ServeMetrics() {
	// Expose prometheus metrics via HTTP.
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/threads", noCache(http.HandlerFunc(goRoutineHandler)))
	klog.Infof("Serving metrics on :%d/metrics", *metricsPort)
	err := http.ListenAndServe(fmt.Sprintf(":%d", *metricsPort), nil)
	if err != nil {
		klog.Fatalf("HTTP ListenAndServe for metrics: %+v", err)
	}
}
