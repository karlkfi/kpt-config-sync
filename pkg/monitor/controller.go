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

// Package monitor contains the controller for monitoring the state of Nomos on a cluster.
package monitor

import (
	"github.com/google/nomos/clientgen/apis/scheme"
	"github.com/google/nomos/pkg/monitor/clusterconfig"
	"github.com/google/nomos/pkg/monitor/namespaceconfig"
	"github.com/google/nomos/pkg/monitor/state"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManager adds all Controllers to the Manager
func AddToManager(mgr manager.Manager) error {
	if err := scheme.AddToScheme(mgr.GetScheme()); err != nil {
		return errors.Wrapf(err, "pkg/monitor.AddToManager")
	}
	cs := state.NewClusterState()

	if err := clusterconfig.AddController(mgr, cs); err != nil {
		return err
	}
	if err := namespaceconfig.AddController(mgr, cs); err != nil {
		return err
	}
	return nil
}
