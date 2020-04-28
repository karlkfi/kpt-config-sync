// Package client contains an enhanced client.
package client

import (
	"context"
	"fmt"
	"time"

	"github.com/davecgh/go-spew/spew"
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
	c.recordLatency(start, "Create", obj.GroupVersionKind().Kind, metrics.StatusLabel(err))

	if err != nil {
		return status.ResourceWrap(err, "failed to create "+description, ast.ParseFileObject(obj))
	}
	glog.Infof("Created %s", description)
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
			glog.V(2).Infof("Delete skipped, %s does not exist", description)
			return nil
		}
		// TODO(b/155203803): determine if this belongs in the non error path
		if isFinalizing(obj) {
			glog.V(2).Infof("Delete skipped, resource is finalizing %s", description)
			return nil
		}
		return errors.Wrapf(err, "failed to get %s during delete operation (this is an API server error)", description)
	}

	start := time.Now()
	opts = append(opts, client.PropagationPolicy(metav1.DeletePropagationBackground))
	err := c.Client.Delete(ctx, obj, opts...)

	if err == nil {
		glog.Infof("Deleted %s", description)
	} else if apierrors.IsNotFound(err) {
		glog.V(2).Infof("Not found during attempted delete %s", description)
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

	for tryNum := 0; tryNum < c.MaxTries; tryNum++ {
		if err := c.Client.Get(ctx, namespacedName, workingObj); err != nil {
			return nil, status.MissingResourceWrap(err, "failed to update "+description, ast.ParseFileObject(obj))
		}
		oldV := resourceVersion(workingObj)
		newObj, err := updateFn(core.DeepCopy(workingObj))
		if err != nil {
			if isNoUpdateNeeded(err) {
				glog.V(2).Infof("Update function for %s returned no update needed", description)
				return newObj, nil
			}
			return nil, status.ResourceWrap(err, "failed to update "+description, ast.ParseFileObject(obj))
		}

		// cmp.Diff may take a while on the resource, only compute if V(1)
		if glog.V(1) {
			glog.Infof("Updating object %q attempt=%d diff old..new:\n%v",
				description, tryNum+1, cmp.Diff(workingObj, newObj))
		}

		start := time.Now()
		err = clientUpdateFn(ctx, newObj)
		c.recordLatency(start, "update", obj.GroupVersionKind().Kind, metrics.StatusLabel(err))

		if err == nil {
			newV := resourceVersion(newObj)
			if oldV == newV {
				glog.Warningf("ResourceVersion for %s did not change during update (noop), updateFn should have indicated no update needed", description)
			} else {
				glog.Infof("Updated %s from ResourceVersion %s to %s", description, oldV, newV)
			}
			return newObj, nil
		}
		lastErr = err

		if !apierrors.IsConflict(err) {
			return nil, status.ResourceWrap(err, "failed to update "+description, ast.ParseFileObject(obj))
		}
		glog.V(2).Infof("Conflict during update for %q: %v", description, err)
		time.Sleep(100 * time.Millisecond) // Back off on retry a bit.
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
		return errors.Wrapf(err, "error getting %s for create/update", description)
	}

	glog.V(1).Infof("Will update %s to %s", description, spew.Sdump(obj))
	start := time.Now()
	err := c.Client.Update(ctx, obj)
	c.recordLatency(start, "update", obj.GroupVersionKind().Kind, metrics.StatusLabel(err))

	if err != nil {
		return errors.Wrapf(err, "upsert failed for %s", description)
	}
	glog.Infof("Updated %s via upsert", description)
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
