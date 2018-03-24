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
package action

// Interface represents a CUD action on a kubernetes resource
type Interface interface {
	// Operation returns the type of operation
	Operation() OperationType
	// Execute will execute the operation then return an error on failure
	Execute() error
	// Resource returns the type of resource being operated on
	Resource() string
	// Kind returns the kind of the resource being operated on
	Kind() string
	// Namespace returns the namespace of the resource being operated on
	Namespace() string
	// Group returns the group of the resource being operated on
	Group() string
	// Version returns the version of the resource being operated on
	Version() string
	// Name returns the name of the resource being operated on
	Name() string
	// String representation of this action. It should uniquely identify the resource being modified,
	// (group, version, kind/resource, namespace, name).
	String() string
}
