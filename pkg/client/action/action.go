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
// TODO(briantkennedy): have Operation() method return OperationType
// TODO(briantkennedy): add Kind(), Version() method and specify that String() should return
// information that uniquely identifies the resource being modified, (group, version, kind/resource,
// namespace, name).
type Interface interface {
	// Operation returns the operation name
	Operation() string
	// Execute will execute the operation then return an error on failure
	Execute() error
	// Resource returns the type of resource being operated on
	Resource() string
	// Namespace returns the namespace of the resource being operated on
	Namespace() string
	// String representation of this action
	String() string
}
