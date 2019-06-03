// Package client contains an enhanced client.
package client

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/metrics"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Client extends the controller-runtime client by exporting prometheus metrics and retrying updates.
type Client struct {
	client.Client
	latencyMetric *prometheus.HistogramVec
	MaxTries      int
}

// New returns a new Client.
func New(client client.Client, latencyMetric *prometheus.HistogramVec) *Client {
	return &Client{
		Client:        client,
		MaxTries:      5,
		latencyMetric: latencyMetric,
	}
}

// clientUpdateFn is a Client function signature for updating an entire resource or a resource's status.
type clientUpdateFn func(ctx context.Context, obj runtime.Object) error

// update is a function that updates the state of an API object. The argument is expected to be a copy of the object,
// so no there is no need to worry about mutating the argument when implementing an Update function.
type update func(runtime.Object) (runtime.Object, error)

// Create saves the object obj in the Kubernetes cluster and records prometheus metrics.
func (c *Client) Create(ctx context.Context, obj runtime.Object) status.Error {
	description, kind := resourceInfo(obj)
	glog.V(1).Infof("Creating %s", description)

	start := time.Now()
	err := c.Client.Create(ctx, obj)
	c.recordLatency(start, "create", kind, metrics.StatusLabel(err))

	if err != nil {
		return status.ResourceWrap(err, "failed to create "+description, ast.ParseFileObject(obj))
	}
	glog.V(1).Infof("Create OK for %s", description)
	return nil
}

// Delete deletes the given obj from Kubernetes cluster and records prometheus metrics.
// This automatically sets the propagation policy to always be "Background".
func (c *Client) Delete(ctx context.Context, obj runtime.Object, opts ...client.DeleteOptionFunc) error {
	description, kind := resourceInfo(obj)
	_, namespacedName := metaNamespacedName(obj)

	if err := c.Client.Get(ctx, namespacedName, obj); err != nil {
		if apierrors.IsNotFound(err) {
			// Object is already deleted
			return nil
		}
		if isFinalizing(obj.(metav1.Object)) {
			glog.V(3).Infof("Delete skipped, resource is finalizing %s", description)
			return nil
		}
		return errors.Wrapf(err, "could not look up object we're deleting %s", description)
	}

	start := time.Now()
	opts = append(opts, client.PropagationPolicy(metav1.DeletePropagationBackground))
	err := c.Client.Delete(ctx, obj, opts...)

	if err == nil {
		glog.V(1).Infof("Delete OK for %s", description)
	} else if apierrors.IsNotFound(err) {
		glog.V(3).Infof("Not found during attempted delete %s", description)
		err = nil
	} else {
		err = errors.Wrapf(err, "delete failed for %s", description)
	}

	c.recordLatency(start, "delete", kind, metrics.StatusLabel(err))
	return nil
}

// Update updates the given obj in the Kubernetes cluster.
func (c *Client) Update(ctx context.Context, obj runtime.Object, updateFn update) (runtime.Object, status.Error) {
	return c.update(ctx, obj, updateFn, c.Client.Update)
}

// UpdateStatus updates the given obj's status in the Kubernetes cluster.
func (c *Client) UpdateStatus(ctx context.Context, obj runtime.Object, updateFn update) (runtime.Object, status.Error) {
	return c.update(ctx, obj, updateFn, c.Client.Status().Update)
}

// update updates the given obj in the Kubernetes cluster using clientUpdateFn and records prometheus
// metrics. In the event of a conflicting update, it will retry.
// This operation always involves retrieving the resource from API Server before actually updating it.
func (c *Client) update(ctx context.Context, obj runtime.Object, updateFn update,
	clientUpdateFn clientUpdateFn) (runtime.Object, status.Error) {
	// We only want to modify the argument after successfully making an update to API Server.
	workingObj := obj.DeepCopyObject()
	description, kind := resourceInfo(workingObj)
	_, namespacedName := metaNamespacedName(workingObj)

	var lastErr error
	var oldObj runtime.Object

	for tryNum := 0; tryNum < c.MaxTries; tryNum++ {
		if err := c.Client.Get(ctx, namespacedName, workingObj); err != nil {
			return nil, status.MissingResourceWrap(err, "failed to update "+description, ast.ParseFileObject(obj))
		}
		oldV := resourceVersion(workingObj)
		newObj, err := updateFn(workingObj.DeepCopyObject())
		if err != nil {
			if IsNoUpdateNeeded(err) {
				return newObj, nil
			}
			return nil, status.ResourceWrap(err, "failed to update "+description, ast.ParseFileObject(obj))
		}

		if glog.V(3) {
			glog.Warningf("update: %q: try: %v diff old..new:\n%v",
				namespacedName, tryNum, cmp.Diff(workingObj, newObj))
			if oldObj != nil {
				glog.Warningf("update: %q: prev..old:\n%v",
					namespacedName, cmp.Diff(oldObj, workingObj))
			}
		}

		start := time.Now()
		err = clientUpdateFn(ctx, newObj)
		c.recordLatency(start, "update", kind, metrics.StatusLabel(err))

		if err == nil {
			newV := resourceVersion(newObj)
			if oldV == newV {
				glog.V(3).Infof("Update not needed for %s", description)
			} else {
				glog.V(1).Infof("Update OK for %s from ResourceVersion %s to %s", description, oldV, newV)
			}
			return newObj, nil
		}
		lastErr = err

		if glog.V(3) {
			glog.Warningf("error in clientUpdateFn(...) for %q: %v", namespacedName, err)
			// Skip the expensive copy if we're not going to use it.
			oldObj = workingObj.DeepCopyObject()
		}
		if !apierrors.IsConflict(err) {
			return nil, status.ResourceWrap(err, "failed to update "+description, ast.ParseFileObject(obj))
		}
		<-time.After(100 * time.Millisecond) // Back off on retry a bit.
	}
	return nil, status.ResourceWrap(lastErr, "exceeded max tries to update "+description, ast.ParseFileObject(obj))
}

// Upsert creates or updates the given obj in the Kubernetes cluster and records prometheus metrics.
// This operation always involves retrieving the resource from API Server before actually creating or updating it.
func (c *Client) Upsert(ctx context.Context, obj runtime.Object) error {
	description, kind := resourceInfo(obj)
	_, namespacedName := metaNamespacedName(obj)
	if err := c.Client.Get(ctx, namespacedName, obj.DeepCopyObject()); err != nil {
		if apierrors.IsNotFound(err) {
			return c.Create(ctx, obj)
		}
		return errors.Wrapf(err, "could not get status of %s while upserting", description)
	}

	start := time.Now()
	err := c.Client.Update(ctx, obj)
	c.recordLatency(start, "update", kind, metrics.StatusLabel(err))

	if err != nil {
		return errors.Wrapf(err, "upsert failed for %s", description)
	}
	glog.V(1).Infof("Upsert OK for %s", description)
	return nil
}

func (c *Client) recordLatency(start time.Time, lvs ...string) {
	if c.latencyMetric == nil {
		return
	}
	c.latencyMetric.WithLabelValues(lvs...).Observe(time.Since(start).Seconds())
}

// resourceInfo returns a description of the object (its GroupVersionKind and NamespacedName), as well as its Kind.
func resourceInfo(obj runtime.Object) (description string, kind string) {
	gvk := obj.GetObjectKind().GroupVersionKind()
	kind = gvk.Kind

	_, n := metaNamespacedName(obj)
	description = fmt.Sprintf("%q, %q", gvk, n)
	return
}

func resourceVersion(obj runtime.Object) string {
	m, _ := metaNamespacedName(obj)
	return m.GetResourceVersion()
}

func metaNamespacedName(obj runtime.Object) (metav1.Object, types.NamespacedName) {
	m := obj.(metav1.Object)
	return m, types.NamespacedName{Namespace: m.GetNamespace(), Name: m.GetName()}
}

// isFinalizing returns true if the object is finalizing.
func isFinalizing(m metav1.Object) bool {
	return m.GetDeletionTimestamp() != nil
}
