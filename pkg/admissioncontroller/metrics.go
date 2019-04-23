/*
Copyright 2018 The CSP Config Management Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
