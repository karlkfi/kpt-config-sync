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

package syncercontroller

import (
	"github.com/google/nomos/pkg/syncer/args"
	"github.com/google/nomos/pkg/syncer/clusterpolicycontroller"
	clustermodules "github.com/google/nomos/pkg/syncer/clusterpolicycontroller/modules"
	"github.com/google/nomos/pkg/syncer/modules"
	"github.com/google/nomos/pkg/syncer/policyhierarchycontroller"
	"github.com/kubernetes-sigs/kubebuilder/pkg/inject/run"
)

// SyncerController sets up the kubebuilder framework
type SyncerController struct {
	injectArgs args.InjectArgs
}

// New returns a new syncer controller with the given inject args.
func New(injectArgs args.InjectArgs) *SyncerController {
	return &SyncerController{
		injectArgs: injectArgs,
	}
}

// Start creates the appropriate sub modules and then starts the controller
func (s *SyncerController) Start(runArgs run.RunArguments) {
	hierarchyModules := []policyhierarchycontroller.Module{
		modules.NewResourceQuotaModule(s.injectArgs.KubernetesClientSet, s.injectArgs.KubernetesInformers),
		modules.NewRoleModule(s.injectArgs.KubernetesClientSet, s.injectArgs.KubernetesInformers),
		modules.NewRoleBindingModule(s.injectArgs.KubernetesClientSet, s.injectArgs.KubernetesInformers),
	}
	s.injectArgs.ControllerManager.AddController(
		policyhierarchycontroller.NewController(s.injectArgs, hierarchyModules))

	clusterModules := []clusterpolicycontroller.Module{
		clustermodules.NewClusterRolesModule(s.injectArgs.KubernetesClientSet, s.injectArgs.KubernetesInformers),
		clustermodules.NewClusterRoleBindingsModule(s.injectArgs.KubernetesClientSet, s.injectArgs.KubernetesInformers),
		clustermodules.NewPodSecurityPoliciesModule(s.injectArgs.KubernetesClientSet, s.injectArgs.KubernetesInformers),
	}
	s.injectArgs.ControllerManager.AddController(
		clusterpolicycontroller.NewController(s.injectArgs, clusterModules))

	s.injectArgs.ControllerManager.RunInformersAndControllers(runArgs)
}
