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

package resourcequota

import (
	"github.com/google/nomos/pkg/api/policyhierarchy"
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics contains the prometheus metric vectors to which the package should record metrics
var Metrics = struct {
	Usage      *prometheus.GaugeVec
	Violations *prometheus.CounterVec
}{
	Usage: prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Help:      "Abstract namespace quota usage per resource type",
			Namespace: policyhierarchy.MetricsNamespace,
			Subsystem: "admission_controller",
			Name:      "usage",
		},
		[]string{"app", "abstractnamespace", "resource"},
	),

	Violations: prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Help:      "Abstract namespace quota violations per resource type",
			Namespace: policyhierarchy.MetricsNamespace,
			Subsystem: "admission_controller",
			Name:      "violations_total",
		},
		[]string{"app", "abstractnamespace", "resource"},
	),
}

func init() {
	prometheus.MustRegister(Metrics.Usage)
	prometheus.MustRegister(Metrics.Violations)
}
