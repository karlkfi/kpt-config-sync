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

package policyimporter

import "github.com/prometheus/client_golang/prometheus"

var Metrics = struct {
	Operations   *prometheus.CounterVec
	Nodes        prometheus.Gauge
	PolicyStates *prometheus.CounterVec
}{
	prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Help:      "Total operations that have been performed to keep policy node hierarchy up-to-date with source of truth",
			Namespace: "nomos",
			Subsystem: "policy_importer",
			Name:      "policy_node_operations_total",
		},
		// e.g. create, update, delete
		[]string{"operation"},
	),
	prometheus.NewGauge(
		prometheus.GaugeOpts{
			Help:      "Number of policy nodes in current state",
			Namespace: "nomos",
			Subsystem: "policy_importer",
			Name:      "policy_nodes",
		},
	),
	prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Help:      "Total number of policy state transitions (A state transition can include changes to multiple resources)",
			Namespace: "nomos",
			Subsystem: "policy_importer",
			Name:      "policy_state_transitions_total",
		},
		// e.g. succeeded, failed
		[]string{"status"},
	),
}

func init() {
	prometheus.MustRegister(Metrics.Operations)
	prometheus.MustRegister(Metrics.Nodes)
	prometheus.MustRegister(Metrics.PolicyStates)
}
