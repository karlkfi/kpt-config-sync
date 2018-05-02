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

package gcp

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/clientgen/informers/externalversions"
	listers_v1 "github.com/google/nomos/clientgen/listers/policyhierarchy/v1"
	watcher "github.com/google/nomos/clientgen/watcher/v1"
	"github.com/google/nomos/pkg/client/meta"
	"github.com/google/nomos/pkg/policyimporter/actions"
	"github.com/google/nomos/pkg/util/policynode"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

const resync = time.Minute * 15

// Controller is a controller for managing Nomos CRDs by importing policies from a filesystem tree.
type Controller struct {
	org                 string
	watcherAddr         string
	client              meta.Interface
	actionFactories     actions.Factories
	informerFactory     externalversions.SharedInformerFactory
	policyNodeLister    listers_v1.PolicyNodeLister
	clusterPolicyLister listers_v1.ClusterPolicyLister
	stopChan            chan struct{}
}

// NewController returns a new Controller.
func NewController(org string, watcherAddr string, client meta.Interface, stopChan chan struct{}) *Controller {
	informerFactory := externalversions.NewSharedInformerFactory(
		client.PolicyHierarchy(), resync)
	f := actions.NewFactories(
		client.PolicyHierarchy().NomosV1(),
		informerFactory.Nomos().V1().PolicyNodes().Lister(),
		informerFactory.Nomos().V1().ClusterPolicies().Lister())

	return &Controller{
		org:                 org,
		watcherAddr:         watcherAddr,
		client:              client,
		actionFactories:     f,
		informerFactory:     informerFactory,
		policyNodeLister:    informerFactory.Nomos().V1().PolicyNodes().Lister(),
		clusterPolicyLister: informerFactory.Nomos().V1().ClusterPolicies().Lister(),
		stopChan:            stopChan,
	}
}

// Run runs the controller and blocks until an error occurs or stopChan is closed. func (c *Controller) Run() error {
func (c *Controller) Run() error {
	// Start informers
	c.informerFactory.Start(c.stopChan)
	glog.Infof("Waiting for cache to sync")
	synced := c.informerFactory.WaitForCacheSync(c.stopChan)
	for syncType, ok := range synced {
		if !ok {
			elemType := syncType.Elem()
			return errors.Errorf("failed to sync %s:%s", elemType.PkgPath(), elemType.Name())
		}
	}
	glog.Infof("Caches synced")

	return c.run()
}

func (c *Controller) run() error {
	glog.Infof("Running")

	currentPolicies, err := policynode.ListPolicies(c.policyNodeLister, c.clusterPolicyLister)
	if err != nil {
		return errors.Wrapf(err, "failed to list current policies")
	}

	// TODO(frankfarzan): Configure auth.
	conn, err := grpc.Dial(c.watcherAddr, grpc.WithInsecure())
	if err != nil {
		return errors.Wrapf(err, "failed to dial to server")
	}
	// nolint: errcheck
	defer conn.Close()
	glog.Infof("Connected to Watcher service at %s", c.watcherAddr)

	client := watcher.NewWatcherClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stream, err := client.Watch(ctx, &watcher.Request{Target: fmt.Sprintf("organizations/%s?recursive=true", c.org)})
	if err != nil {
		return errors.Wrap(err, "failed to start streaming RPC to Watcher API")
	}
	glog.Infof("Started streaming RPC to Watcher API")

	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	go func() {
		<-c.stopChan
		glog.Infof("Stop received, cancelling context")
		cancel2()
	}()
	ch := make(chan actionVal)
	g := actionGenerator{
		stream:          stream,
		out:             ch,
		currentPolicies: *currentPolicies,
		actionFactories: c.actionFactories,
	}
	go g.generate(ctx2)
	return applyActions(ch)
}

func applyActions(ch <-chan actionVal) error {
	for d := range ch {
		if d.err != nil {
			return d.err
		}
		if err := d.action.Execute(); err != nil {
			return err
		}
	}
	return nil
}
