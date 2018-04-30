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
// Reviewed by sunilarora

package actions

// ResourceInterface is here to deal with all the actions being roughly equivalent with the
// exception of a few places.  The main goal of this is to reduce the surface area required for
// performing action tests.
// NOTE: All "interface{}" types passed to this function will be a non-nil pointer to the instance.
type ResourceInterface interface {
	// Values returns a map of name to value for all resources that currently exist in the
	// namespace. If the resource is cluster level, this should return always return the cluster
	// resource values.
	Values(namespace string) (map[string]interface{}, error)
}
