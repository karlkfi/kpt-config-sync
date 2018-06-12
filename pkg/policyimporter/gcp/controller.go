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
	"crypto/tls"
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
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
)

const resync = time.Minute * 15

var scopes = []string{"https://www.googleapis.com/auth/cloud-platform"}

// Controller is a controller for managing Nomos CRDs by importing policies from a filesystem tree.
type Controller struct {
	org                 string
	watcherAddr         string
	credsFile           string
	scopes              []string
	client              meta.Interface
	actionFactories     actions.Factories
	informerFactory     externalversions.SharedInformerFactory
	policyNodeLister    listers_v1.PolicyNodeLister
	clusterPolicyLister listers_v1.ClusterPolicyLister
	stopChan            chan struct{}
}

// NewController returns a new Controller.
func NewController(org, watcherAddr, credsFile string, client meta.Interface, stopChan chan struct{}) *Controller {
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
		credsFile:           credsFile,
		scopes:              scopes,
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

	if c.credsFile == "" {
		return fmt.Errorf("a credentials file is required")
	}

	currentPolicies, listErr := policynode.ListPolicies(c.policyNodeLister, c.clusterPolicyLister)
	if listErr != nil {
		return errors.Wrapf(listErr, "failed to list current policies")
	}

	perRPC, err := oauth.NewServiceAccountFromFile(c.credsFile, c.scopes...)
	if err != nil {
		return errors.Wrapf(err, "while creating credentials from: %q", c.credsFile)
	}
	opts := []grpc.DialOption{
		grpc.WithPerRPCCredentials(perRPC),
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})),
	}

	conn, err := grpc.Dial(c.watcherAddr, opts...)
	if err != nil {
		return errors.Wrapf(err, "failed to dial to server")
	}
	// nolint: errcheck
	defer conn.Close()
	glog.Infof("Connected to Watcher service at %s", c.watcherAddr)

	client := watcher.NewWatcherClient(conn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream, err := client.Watch(ctx, &watcher.Request{Target: fmt.Sprintf("organizations/%s?recursive=true", c.org)})
	if err != nil {
		return errors.Wrap(err, "failed to start streaming RPC to Watcher API")
	}
	glog.Infof("Started streaming RPC to Watcher API")

	go func() {
		<-c.stopChan
		glog.Infof("Stop received, cancelling context")
		cancel()
	}()
	ch := make(chan actionVal)
	g := actionGenerator{
		stream:          stream,
		out:             ch,
		currentPolicies: *currentPolicies,
		actionFactories: c.actionFactories,
	}
	go g.generate(ctx)
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
