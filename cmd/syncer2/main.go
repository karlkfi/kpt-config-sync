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

// Command line util to sync the PolicyNode custom resource to the active namespaces.
package main

import (
	"flag"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/client/informers/externalversions"
	"github.com/google/nomos/pkg/client/policyhierarchy"
	"github.com/pkg/errors"
	core_v1 "k8s.io/api/core/v1"

	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/util/log"
	"github.com/kubernetes-sigs/kubebuilder/pkg/config"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/types"
	"github.com/kubernetes-sigs/kubebuilder/pkg/inject/args"
	"github.com/kubernetes-sigs/kubebuilder/pkg/inject/run"
	"github.com/kubernetes-sigs/kubebuilder/pkg/signals"
)

var resyncPeriond = flag.Duration(
	"resyncPeriod", 2*time.Minute, "resync periond for policyhierarchy")

func main() {
	flag.Parse()
	log.Setup()

	restConfig := config.GetConfigOrDie()
	runArgs := run.RunArguments{Stop: signals.SetupSignalHandler()}

	// create factory / client from restconfig
	client, err := policyhierarchy.NewForConfig(restConfig)
	if err != nil {
		panic(errors.Wrapf(err, "failed to create policyhierarchy clientset"))
	}

	informerFactory := externalversions.NewSharedInformerFactory(client, *resyncPeriond)

	injectArgs := args.CreateInjectArgs(restConfig)

	if err := injectArgs.ControllerManager.AddInformerProvider(
		&policyhierarchy_v1.PolicyNode{},
		informerFactory.Nomos().V1().PolicyNodes()); err != nil {
		panic(err)
	}

	if err := injectArgs.ControllerManager.AddInformerProvider(
		&core_v1.Namespace{},
		injectArgs.KubernetesInformers.Core().V1().Namespaces()); err != nil {
		panic(err)
	}

	injectArgs.ControllerManager.AddController(&controller.GenericController{
		Name:             "syncer-2",
		InformerRegistry: injectArgs.ControllerManager,
		Reconcile: func(k types.ReconcileKey) error {
			glog.Infof("No ReconcileFn defined - skipping %+v", k)
			return nil
		},
	})

	<-runArgs.Stop
}
