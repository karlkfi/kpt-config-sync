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

package metrics

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/prometheus/client_golang/prometheus"
)

// Prometheus metrics
var (
	ErrTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Help:      "Total errors that occurred when executing syncer actions",
			Namespace: configmanagement.MetricsNamespace,
			Subsystem: "syncer",
			Name:      "error_total",
		},
		[]string{"namespace", "resource", "operation"},
	)
	EventTimes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Help:      "Timestamps when syncer events occurred",
			Namespace: configmanagement.MetricsNamespace,
			Subsystem: "syncer",
			Name:      "event_timestamps",
		},
		[]string{"type"},
	)
	ClusterReconcileDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Help:      "Syncer cluster reconciliation duration distributions",
			Namespace: configmanagement.MetricsNamespace,
			Subsystem: "syncer",
			Name:      "syncer_clusterconfig_reconcile_seconds",
			Buckets:   []float64{.001, .01, .1, 1, 10, 100},
		},
		nil,
	)
	NamespaceReconcileDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Help:      "Syncer namespace reconcile duration distributions",
			Namespace: configmanagement.MetricsNamespace,
			Subsystem: "syncer",
			Name:      "namespace_reconcile_duration_seconds",
			Buckets:   []float64{.001, .01, .1, 1, 10, 100},
		},
		[]string{"namespace"},
	)
)

func init() {
	prometheus.MustRegister(
		ErrTotal,
		EventTimes,
		ClusterReconcileDuration,
		NamespaceReconcileDuration,
	)
}
