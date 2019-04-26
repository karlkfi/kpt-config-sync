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

package importer

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics contains the Prometheus metrics for the Importer.
var Metrics = struct {
	APICallDuration  *prometheus.HistogramVec
	CycleDuration    *prometheus.HistogramVec
	NamespaceConfigs prometheus.Gauge
	Operations       *prometheus.CounterVec
}{
	APICallDuration: prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Help:      "Distribution of durations of API server calls",
			Namespace: configmanagement.MetricsNamespace,
			Subsystem: "importer",
			Name:      "api_duration_seconds",
			Buckets:   []float64{.001, .01, .1, 1},
		},
		// operation: create, update, delete
		// type: namespace, cluster, sync
		// status: success, error
		[]string{"operation", "type", "status"},
	),
	CycleDuration: prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Help:      "Distribution of durations of cycles that the importer has attempted to complete",
			Namespace: configmanagement.MetricsNamespace,
			Subsystem: "importer",
			Name:      "cycle_duration_seconds",
		},
		// status: success, error
		[]string{"status"},
	),
	Operations: prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Help:      "Total operations that have been performed to keep configs up-to-date with source of truth",
			Namespace: configmanagement.MetricsNamespace,
			Subsystem: "importer",
			Name:      "operations_total",
		},
		// operation: create, update, delete
		// type: namespace, cluster, sync
		// status: success, error
		[]string{"operation", "type", "status"},
	),
	NamespaceConfigs: prometheus.NewGauge(
		prometheus.GaugeOpts{
			Help:      "Number of namespace configs present in current state",
			Namespace: configmanagement.MetricsNamespace,
			Subsystem: "importer",
			Name:      "namespace_configs",
		},
	),
}

func init() {
	prometheus.MustRegister(
		Metrics.APICallDuration,
		Metrics.CycleDuration,
		Metrics.NamespaceConfigs,
		Metrics.Operations,
	)
}
