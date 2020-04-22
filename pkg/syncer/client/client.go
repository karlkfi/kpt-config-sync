// Package client contains an enhanced client.
package client

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/core"
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
type clientUpdateFn func(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error

// update is a function that updates the state of an API object. The argument is expected to be a copy of the object,
// so no there is no need to worry about mutating the argument when implementing an Update function.
type update func(core.Object) (core.Object, error)

// Create saves the object obj in the Kubernetes cluster and records prometheus metrics.
func (c *Client) Create(ctx context.Context, obj core.Object) status.Error {
	description := getResourceInfo(obj)
	glog.V(1).Infof("Creating %s", description)

	start := time.Now()
	err := c.Client.Create(ctx, obj)
	c.recordLatency(start, "create", obj.GroupVersionKind().Kind, metrics.StatusLabel(err))

	if err != nil {
		return status.ResourceWrap(err, "failed to create "+description, ast.ParseFileObject(obj))
	}
	glog.V(1).Infof("Create OK for %s", description)
	return nil
}

// Delete deletes the given obj from Kubernetes cluster and records prometheus metrics.
// This automatically sets the propagation policy to always be "Background".
func (c *Client) Delete(ctx context.Context, obj core.Object, opts ...client.DeleteOption) error {
	description := getResourceInfo(obj)
	namespacedName := getNamespacedName(obj)

	if err := c.Client.Get(ctx, namespacedName, obj); err != nil {
		if apierrors.IsNotFound(err) {
			// Object is already deleted
			return nil
		}
		if isFinalizing(obj) {
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

	c.recordLatency(start, "delete", obj.GroupVersionKind().Kind, metrics.StatusLabel(err))
	return nil
}

// Update updates the given obj in the Kubernetes cluster.
func (c *Client) Update(ctx context.Context, obj core.Object, updateFn update) (runtime.Object, status.Error) {
	return c.update(ctx, obj, updateFn, c.Client.Update)
}

// UpdateStatus updates the given obj's status in the Kubernetes cluster.
func (c *Client) UpdateStatus(ctx context.Context, obj core.Object, updateFn update) (runtime.Object, status.Error) {
	return c.update(ctx, obj, updateFn, c.Client.Status().Update)
}

// update updates the given obj in the Kubernetes cluster using clientUpdateFn and records prometheus
// metrics. In the event of a conflicting update, it will retry.
// This operation always involves retrieving the resource from API Server before actually updating it.
func (c *Client) update(ctx context.Context, obj core.Object, updateFn update,
	clientUpdateFn clientUpdateFn) (runtime.Object, status.Error) {
	// We only want to modify the argument after successfully making an update to API Server.
	workingObj := core.DeepCopy(obj)
	description := getResourceInfo(workingObj)
	namespacedName := getNamespacedName(workingObj)

	var lastErr error
	var oldObj core.Object

	for tryNum := 0; tryNum < c.MaxTries; tryNum++ {
		if err := c.Client.Get(ctx, namespacedName, workingObj); err != nil {
			return nil, status.MissingResourceWrap(err, "failed to update "+description, ast.ParseFileObject(obj))
		}
		oldV := resourceVersion(workingObj)
		newObj, err := updateFn(core.DeepCopy(workingObj))
		if err != nil {
			if isNoUpdateNeeded(err) {
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
		c.recordLatency(start, "update", obj.GroupVersionKind().Kind, metrics.StatusLabel(err))

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
			oldObj = workingObj
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
func (c *Client) Upsert(ctx context.Context, obj core.Object) error {
	description := getResourceInfo(obj)
	namespacedName := getNamespacedName(obj)
	if err := c.Client.Get(ctx, namespacedName, obj.DeepCopyObject()); err != nil {
		if apierrors.IsNotFound(err) {
			return c.Create(ctx, obj)
		}
		return errors.Wrapf(err, "could not get status of %s while upserting", description)
	}

	start := time.Now()
	err := c.Client.Update(ctx, obj)
	c.recordLatency(start, "update", obj.GroupVersionKind().Kind, metrics.StatusLabel(err))

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

// getResourceInfo returns a description of the object (its GroupVersionKind and NamespacedName), as well as its Kind.
func getResourceInfo(obj core.Object) string {
	gvk := obj.GroupVersionKind()
	namespacedName := getNamespacedName(obj)
	return fmt.Sprintf("%q, %q", gvk, namespacedName)
}

func getNamespacedName(obj core.Object) types.NamespacedName {
	return types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}
}

func resourceVersion(obj core.Object) string {
	return obj.GetResourceVersion()
}

type hasDeletionTimestamp interface {
	GetDeletionTimestamp() *metav1.Time
}

// isFinalizing returns true if the object is finalizing.
func isFinalizing(o core.Object) bool {
	return o.(hasDeletionTimestamp).GetDeletionTimestamp() != nil
}
