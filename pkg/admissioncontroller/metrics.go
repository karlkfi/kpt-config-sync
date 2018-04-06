/*
Copyright 2018 The Nomos Authors.
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

import "github.com/prometheus/client_golang/prometheus"

var Metrics = struct {
	AdmitDuration *prometheus.HistogramVec
	ErrorTotal    *prometheus.CounterVec
}{
	prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Help:      "Admission duration distributions",
			Namespace: "nomos",
			Subsystem: "admission_controller",
			Name:      "duration_seconds",
			Buckets:   []float64{.001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 2.5},
		},
		[]string{"app", "namespace", "allowed"},
	),
	prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Help:      "Total internal errors that occurred when reviewing admission requests",
			Namespace: "nomos",
			Subsystem: "admission_controller",
			Name:      "error_total",
		},
		[]string{"app", "namespace"},
	),
}

func init() {
	prometheus.MustRegister(Metrics.AdmitDuration)
	prometheus.MustRegister(Metrics.ErrorTotal)
}
