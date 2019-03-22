/*
Copyright 2017 The CSP Config Management Authors.
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

package action

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/prometheus/client_golang/prometheus"
)

// Actions is a counter for the number of actions executed.
var Actions = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Help:      "The total count of actions created",
		Namespace: configmanagement.MetricsNamespace,
		Subsystem: "action",
		Name:      "executed",
	},
	[]string{"resource", "operation"},
)

// APICalls is a counter for actual api calls
var APICalls = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Help:      "The total count of actual API calls (actions will elide noop API calls)",
		Namespace: configmanagement.MetricsNamespace,
		Subsystem: "action",
		Name:      "api_calls",
	},
	[]string{"resource", "operation"},
)

// APICallDuration tracks the amount of time API calls take.
var APICallDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Help:      "Client action duration distributions",
		Namespace: configmanagement.MetricsNamespace,
		Subsystem: "action",
		Name:      "api_duration_seconds",
		Buckets:   []float64{.001, .01, .1, 1},
	},
	[]string{"resource", "operation"},
)

func init() {
	prometheus.MustRegister(
		Actions,
		APICalls,
		APICallDuration,
	)
}
