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

package syncercontroller

import (
	"fmt"
	"net/http"
	"time"

	"net"

	"github.com/golang/glog"
	policyhierarchyscheme "github.com/google/nomos/clientgen/policyhierarchy/scheme"
	"github.com/google/nomos/pkg/syncer/args"
	"github.com/google/nomos/pkg/syncer/clusterpolicycontroller"
	clustermodules "github.com/google/nomos/pkg/syncer/clusterpolicycontroller/modules"
	"github.com/google/nomos/pkg/syncer/modules"
	"github.com/google/nomos/pkg/syncer/policyhierarchycontroller"
	"github.com/kubernetes-sigs/kubebuilder/pkg/inject/run"
	"github.com/pkg/errors"
	"github.com/prometheus/common/expfmt"
	"k8s.io/client-go/kubernetes/scheme"
)

// importerTimeout is the time to wait for the Importer to come up before the syncer is started
const importerTimeout = time.Second * 30

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

// Wait for Importer waits for the importer to come up, and returns
// an error if timeout is reached or an error is encountered. This is necessary
// because otherwise the syncer may start to sync an empty state, which can
// lead to the destruction of namespaces managed by a previous installation.
func (s *SyncerController) waitForImporter(timeout time.Duration) error {
	for t := time.Now(); time.Since(t) < timeout; time.Sleep(time.Second) {
		var target string
		if s.injectArgs.SyncerOptions.GCPMode {
			target = "http://gcp-policy-importer:8675/metrics"
		} else {
			target = "http://git-policy-importer:8675/metrics"
		}

		c := http.Client{
			Timeout: 5 * time.Second,
		}
		resp, err := c.Get(target)
		if err, ok := err.(net.Error); ok && err.Timeout() {
			glog.Infof("timed out trying to access importer metrics. Trying again. Error was %v", err)
			continue
		} else if dnsErr, ok := err.(*net.DNSError); ok {
			glog.Warningf("DNS error while trying to contact. Trying again. Error was: %v", dnsErr)
			continue
		} else if err != nil {
			return errors.Wrap(err, "failed while checking importer readiness")
		}

		var parser expfmt.TextParser
		mf, err := parser.TextToMetricFamilies(resp.Body)
		if err != nil {
			resp.Body.Close()
			return errors.Wrap(err, "failed while parsing metrics")
		}
		metrics := mf["nomos_policy_importer_policy_state_transitions_total"].GetMetric()
		for _, m := range metrics {
			labels := m.GetLabel()
			for _, l := range labels {
				if *l.Name == "status" && *l.Value == "succeeded" && m.Counter.GetValue() > 0 {
					glog.Info("initial importer sync completed")
					resp.Body.Close()
					return nil
				}
			}
		}
		resp.Body.Close()
	}
	return fmt.Errorf("timed out waiting for importer to sync state")
}

// Start creates the appropriate sub modules and then starts the controller
func (s *SyncerController) Start(runArgs run.RunArguments) error {
	err := s.waitForImporter(importerTimeout)
	if err != nil {
		return fmt.Errorf("failed waiting for importer: %v", err)
	}

	policyhierarchyscheme.AddToScheme(scheme.Scheme)

	hierarchyModules := []policyhierarchycontroller.Module{
		modules.NewRole(s.injectArgs.KubernetesClientSet, s.injectArgs.KubernetesInformers),
		modules.NewRoleBinding(s.injectArgs.KubernetesClientSet, s.injectArgs.KubernetesInformers),
	}
	clusterModules := []clusterpolicycontroller.Module{
		clustermodules.NewClusterRoles(s.injectArgs.KubernetesClientSet, s.injectArgs.KubernetesInformers),
		clustermodules.NewClusterRoleBindings(s.injectArgs.KubernetesClientSet, s.injectArgs.KubernetesInformers),
	}

	if !s.injectArgs.SyncerOptions.GCPMode {
		hierarchyModules = append(
			hierarchyModules,
			modules.NewResourceQuota(s.injectArgs.KubernetesClientSet, s.injectArgs.KubernetesInformers))
		clusterModules = append(
			clusterModules,
			clustermodules.NewPodSecurityPolicies(s.injectArgs.KubernetesClientSet, s.injectArgs.KubernetesInformers))
	}

	s.injectArgs.ControllerManager.AddController(
		policyhierarchycontroller.NewController(s.injectArgs, hierarchyModules))
	s.injectArgs.ControllerManager.AddController(
		clusterpolicycontroller.NewController(s.injectArgs, clusterModules))
	s.injectArgs.ControllerManager.RunInformersAndControllers(runArgs)
	return nil
}
