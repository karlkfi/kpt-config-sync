/*
Copyright 2017 The Nomos Authors.

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

package policyhierarchy

const (
	// CLIName is the short name of the CLI.
	CLIName = "nomos"

	// MetricsNamespace is the namespace that metrics are held in.
	MetricsNamespace = "gkeconfig"

	// OperatorKind is the Kind of the Operator object.
	OperatorKind = "Nomos"

	// GroupName is the name of the group of policyhierarchy resources.
	GroupName = "configmanagement.gke.io"

	// ProductName is what we call Nomos externally.
	ProductName = "CSP Configuration Management"

	// ControllerNamespace is the Namespace used for Nomos controllers
	ControllerNamespace = "config-management-system"
)
