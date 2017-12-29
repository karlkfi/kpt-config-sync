/*
Copyright 2017 The Stolos Authors.

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

package registration

import (
	"github.com/google/stolos/pkg/cli"
	"github.com/google/stolos/pkg/cli/namespaces"
	"github.com/google/stolos/pkg/cli/quota"
	"github.com/google/stolos/pkg/cli/rolebindings"
)

func init() {
	// Register CLI commands here, try to keep these alphabetized.
	cli.RegisterKubectlPluginFunction([]string{"get", "namespaces"}, namespaces.GetHierarchicalNamespaces)
	cli.RegisterKubectlPluginFunction([]string{"get", "quota"}, quota.GetHierarchical)
	cli.RegisterKubectlPluginFunction([]string{"get", "rolebindings"}, rolebindings.GetHierarchicalRoleBindings)
	cli.RegisterKubectlPluginFunction([]string{"get", "roles"}, rolebindings.GetHierarchicalRoles)
}
