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

// Package args contains kubebuilder InjectArgs for monitor controller.
package args

import (
	"time"

	"github.com/google/nomos/clientgen/apis"
	informers "github.com/google/nomos/clientgen/informer"
	"github.com/kubernetes-sigs/kubebuilder/pkg/inject/args"
	"k8s.io/client-go/rest"
)

// InjectArgs contains the arguments need to initialize monitor controller.
type InjectArgs struct {
	args.InjectArgs // kubebuilder inject args

	Informers informers.SharedInformerFactory
}

// CreateInjectArgs returns new controller args.
func CreateInjectArgs(config *rest.Config) InjectArgs {
	client := apis.NewForConfigOrDie(config)
	return InjectArgs{
		InjectArgs: args.CreateInjectArgs(config),
		Informers:  informers.NewSharedInformerFactory(client, time.Minute),
	}
}
