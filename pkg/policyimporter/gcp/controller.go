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
	"crypto/x509"
	"fmt"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	"github.com/google/nomos/clientgen/informers/externalversions"
	listers_v1 "github.com/google/nomos/clientgen/listers/policyhierarchy/v1"
	policyhierarchyscheme "github.com/google/nomos/clientgen/policyhierarchy/scheme"
	watcher "github.com/google/nomos/clientgen/watcher/v1"
	client_action "github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/client/meta"
	"github.com/google/nomos/pkg/policyimporter/actions"
	"github.com/google/nomos/pkg/util/policynode"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/grpc/keepalive"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/util/cert"
)

const (
	// Timeout for making initial grpc connection dial.
	grpcDialTimeout = time.Second * 20
	// After a duration of this time if the client doesn't see any activity it pings the server to see if the transport is still alive.
	grpcKeepaliveTime = time.Minute
	// After having pinged for keepalive check, the client waits this long and if no activity is seen even after that
	// the connection is closed. (This will eagerly fail inflight grpc requests even if they don't have timeouts.)
	grpcKeepaliveTimeout = time.Minute
	// Resync period for K8S informer.
	informerResync = time.Minute * 15
)

var scopes = []string{"https://www.googleapis.com/auth/cloud-platform"}

// Controller is a controller for managing Nomos CRDs by importing policies from a filesystem tree.
type Controller struct {
	org                 string
	watcherAddr         string
	credsFile           string
	caFile              string
	scopes              []string
	client              meta.Interface
	actionFactories     actions.Factories
	informerFactory     externalversions.SharedInformerFactory
	policyNodeLister    listers_v1.PolicyNodeLister
	clusterPolicyLister listers_v1.ClusterPolicyLister
	stopChan            chan struct{}
}

// NewController returns a new Controller.
func NewController(org, watcherAddr, credsFile, caFile string, client meta.Interface, stopChan chan struct{}) *Controller {
	policyhierarchyscheme.AddToScheme(scheme.Scheme)

	informerFactory := externalversions.NewSharedInformerFactory(
		client.PolicyHierarchy(), informerResync)
	f := actions.NewFactories(
		client.PolicyHierarchy().NomosV1(),
		informerFactory.Nomos().V1().PolicyNodes().Lister(),
		informerFactory.Nomos().V1().ClusterPolicies().Lister())

	return &Controller{
		org:                 org,
		watcherAddr:         watcherAddr,
		client:              client,
		credsFile:           credsFile,
		caFile:              caFile,
		scopes:              scopes,
		actionFactories:     f,
		informerFactory:     informerFactory,
		policyNodeLister:    informerFactory.Nomos().V1().PolicyNodes().Lister(),
		clusterPolicyLister: informerFactory.Nomos().V1().ClusterPolicies().Lister(),
		stopChan:            stopChan,
	}
}

// Run runs the controller and blocks until an error occurs or stopChan is closed.
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

	return c.watchLoop()
}

// Executes startWatch() in a loop, returning if it fails too many times or stopChan is closed.
func (c *Controller) watchLoop() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-c.stopChan
		glog.Infof("Stop received, cancelling context")
		cancel()
	}()

	var resumeMarker []byte
	throttler := time.NewTicker(time.Second)
	defer throttler.Stop()

	for {
		select {
		case <-c.stopChan:
			glog.Infof("Stop received, cancelling watch loop")
			return nil
		default:
			var b backoff.BackOff
			b = &backoff.ExponentialBackOff{
				InitialInterval:     2 * time.Second,
				RandomizationFactor: 0.5,
				Multiplier:          2,
				MaxInterval:         5 * time.Minute,
				MaxElapsedTime:      0,
				Clock:               backoff.SystemClock,
			}
			b = backoff.WithContext(b, ctx)
			b = backoff.WithMaxRetries(b, 5)

			err := backoff.RetryNotify(func() error {
				rm, err := c.watchIteration(ctx, resumeMarker)
				if rm != nil {
					resumeMarker = rm
				}
				return err
			},
				b,
				func(err error, next time.Duration) {
					glog.Warningf("error: retrying in %v on error %v", next, err)
				})

			// Maximum number of retries hit, fail out
			if err != nil {
				return err
			}

			// Throttle all reconnections to occur at most once per second
			// This part of the loop should only be hit on normal connection terminations (i.e.
			// backoff didn't return an error)
			<-throttler.C
		}
	}
}

// Executes the Watch API with the given resume marker and returns any errors and a resume marker.
func (c *Controller) watchIteration(ctx context.Context, resumeMarker []byte) ([]byte, error) {
	glog.Infof("Starting watch")

	currentPolicies, listErr := policynode.ListPolicies(c.policyNodeLister, c.clusterPolicyLister)
	if listErr != nil {
		return nil, errors.Wrapf(listErr, "failed to list current policies")
	}

	conn, err := c.dial()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to dial to server")
	}
	defer func() {
		if err2 := conn.Close(); err2 != nil {
			glog.Errorf("Failed to close connection: %v", err2)
		}
	}()
	glog.Infof("Connected to Watcher service at %s", c.watcherAddr)

	client := watcher.NewWatcherClient(conn)
	ctx, cancelWatch := context.WithCancel(ctx)
	defer cancelWatch()
	stream, err := client.Watch(ctx, &watcher.Request{Target: fmt.Sprintf("organizations/%s?recursive=true", c.org), ResumeMarker: resumeMarker})
	if err != nil {
		return nil, errors.Wrap(err, "failed to start streaming RPC to Watcher API")
	}
	glog.Infof("Started streaming RPC to Watcher API")

	return newWatchProcessor(stream, applyActions, *currentPolicies, c.actionFactories, len(resumeMarker) != 0).process()
}

func (c *Controller) dial() (*grpc.ClientConn, error) {
	perRPC, err := oauth.NewServiceAccountFromFile(c.credsFile, c.scopes...)
	if err != nil {
		return nil, errors.Wrapf(err, "while creating credentials from: %q", c.credsFile)
	}
	config, err := tlsConfig(c.caFile)
	if err != nil {
		return nil, errors.Wrapf(err, "while creating TLS config")
	}
	opts := []grpc.DialOption{
		grpc.WithPerRPCCredentials(perRPC),
		grpc.WithTransportCredentials(credentials.NewTLS(config)),
		grpc.WithBlock(),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{Time: grpcKeepaliveTime, Timeout: grpcKeepaliveTimeout}),
	}

	ctx, cancel := context.WithTimeout(context.Background(), grpcDialTimeout)
	defer cancel()
	return grpc.DialContext(ctx, c.watcherAddr, opts...)
}

// tlsConfig returns a TLS configuration from a supplied CA certificate file,
// if the file is not supplied (empty), the config defaults to using the system
// root CA certificate file.  Taken from the K8S oidc implementation.
func tlsConfig(caFile string) (*tls.Config, error) {
	var roots *x509.CertPool
	if caFile != "" {
		var err error
		roots, err = cert.NewPool(caFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read the CA file %q: %v", caFile, err)
		}
		glog.Infof("Loaded CA cert file: %q", caFile)
	} else {
		glog.Info("Using host CA certificates set.")
	}
	config := &tls.Config{
		// If certpool is nil (caFile unset), this will default to using the
		// system CA roots.
		RootCAs: roots,
	}
	return config, nil
}

func applyActions(action client_action.Interface) error {
	return action.Execute()
}
