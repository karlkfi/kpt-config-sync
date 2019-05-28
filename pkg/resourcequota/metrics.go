package resourcequota

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics contains the prometheus metric vectors to which the package should record metrics
var Metrics = struct {
	Usage      *prometheus.GaugeVec
	Violations *prometheus.CounterVec
}{
	Usage: prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Help:      "Quota usage per resource type",
			Namespace: configmanagement.MetricsNamespace,
			Subsystem: "admission_controller",
			Name:      "usage",
		},
		[]string{"resource"},
	),

	Violations: prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Help:      "Quota violations per resource type",
			Namespace: configmanagement.MetricsNamespace,
			Subsystem: "admission_controller",
			Name:      "violations_total",
		},
		[]string{"resource"},
	),
}

func init() {
	prometheus.MustRegister(Metrics.Usage)
	prometheus.MustRegister(Metrics.Violations)
}
