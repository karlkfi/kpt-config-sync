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
// Reviewed by sunilarora

package args

import (
	informers "github.com/google/nomos/clientgen/informers/policyhierarchy"
	"github.com/google/nomos/clientgen/policyhierarchy"
	"github.com/google/nomos/pkg/syncer/options"
	"github.com/kubernetes-sigs/kubebuilder/pkg/inject/args"
	"k8s.io/client-go/rest"
)

// InjectArgs are the arguments need to initialize controllers
type InjectArgs struct {
	args.InjectArgs // kubebuilder inject args

	Clientset     *policyhierarchy.Clientset
	Informers     informers.SharedInformerFactory
	SyncerOptions options.Options
}

// CreateInjectArgs returns new controller args
func CreateInjectArgs(config *rest.Config) InjectArgs {
	client := policyhierarchy.NewForConfigOrDie(config)
	syncerOptions := options.FromFlagsAndEnv()
	return InjectArgs{
		InjectArgs:    args.CreateInjectArgs(config),
		Clientset:     client,
		Informers:     informers.NewSharedInformerFactory(client, syncerOptions.ResyncPeriod),
		SyncerOptions: syncerOptions,
	}
}
