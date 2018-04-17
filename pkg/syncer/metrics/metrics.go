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

package metrics

import "github.com/prometheus/client_golang/prometheus"

// Prometheus metrics
var (
	ErrTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Help:      "Total errors that occurred when executing syncer actions",
			Namespace: "nomos",
			Subsystem: "syncer",
			Name:      "error_total",
		},
		[]string{"namespace", "resource", "operation"},
	)
	EventTimes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Help:      "Timestamps when syncer events occurred",
			Namespace: "nomos",
			Subsystem: "syncer",
			Name:      "event_timestamps",
		},
		[]string{"type"},
	)
	QueueSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Help:      "Current size of syncer action queue",
			Namespace: "nomos",
			Subsystem: "syncer",
			Name:      "queue_size",
		})
	ClusterReconcileDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Help:      "Syncer cluster reconciliation duration distributions",
			Namespace: "nomos",
			Subsystem: "syncer",
			Name:      "syncer_clusterpolicy_reconcile_seconds",
			Buckets:   []float64{.001, .01, .1, 1, 10, 100},
		},
		nil,
	)
	HierarchicalReconcileDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Help:      "Syncer hierarchical reconcile duration distributions",
			Namespace: "nomos",
			Subsystem: "syncer",
			Name:      "hierarchical_reconcile_duration_seconds",
			Buckets:   []float64{.001, .01, .1, 1, 10, 100},
		},
		[]string{"namespace"},
	)
)

func init() {
	prometheus.MustRegister(
		ErrTotal,
		EventTimes,
		QueueSize,
		HierarchicalReconcileDuration,
	)
}
