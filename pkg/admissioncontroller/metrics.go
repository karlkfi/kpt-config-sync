package admissioncontroller

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics contains the prometheus metric vectors to which the package should record metrics
var Metrics = struct {
	AdmitDuration *prometheus.HistogramVec
	ErrorTotal    prometheus.Counter
}{
	prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Help:      "Admission duration distributions",
			Namespace: configmanagement.MetricsNamespace,
			Subsystem: "admission_controller",
			Name:      "duration_seconds",
			Buckets:   []float64{.001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 2.5},
		},
		[]string{"allowed"},
	),
	prometheus.NewCounter(
		prometheus.CounterOpts{
			Help:      "Total internal errors that occurred when reviewing admission requests",
			Namespace: configmanagement.MetricsNamespace,
			Subsystem: "admission_controller",
			Name:      "error_total",
		},
	),
}

func init() {
	prometheus.MustRegister(
		Metrics.AdmitDuration,
		Metrics.ErrorTotal,
	)
}
