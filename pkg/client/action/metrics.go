package action

import "github.com/prometheus/client_golang/prometheus"

var Duration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Help:      "Client action duration distributions",
		Namespace: "nomos",
		Subsystem: "action",
		Name:      "duration_seconds",
		Buckets:   []float64{.001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 2.5},
	},
	[]string{"namespace", "resource", "operation"},
)

func init() {
	prometheus.MustRegister(Duration)
}
