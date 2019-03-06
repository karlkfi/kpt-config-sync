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

// Package client contains an enhanced client.
package client

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/client/action"
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
	MaxTries int
}

// New returns a new Client.
func New(client client.Client) *Client {
	return &Client{
		Client:   client,
		MaxTries: 5,
	}
}

// clientUpdateFn is a Client function signature for updating an entire resource or a resource's status.
type clientUpdateFn func(ctx context.Context, obj runtime.Object) error

// Create saves the object obj in the Kubernetes cluster and records prometheus metrics.
func (c *Client) Create(ctx context.Context, obj runtime.Object) error {
	description, kind := resourceInfo(obj)
	glog.V(1).Infof("Creating %s", description)
	operation := string(action.CreateOperation)
	action.Actions.WithLabelValues(kind, operation).Inc()
	action.APICalls.WithLabelValues(kind, operation).Inc()
	timer := prometheus.NewTimer(action.APICallDuration.WithLabelValues(kind, operation))
	defer timer.ObserveDuration()
	if err := c.Client.Create(ctx, obj); err != nil {
		return errors.Wrapf(err, "failed to create %s", description)
	}
	glog.V(1).Infof("Create OK for %s", description)
	return nil
}

// Delete deletes the given obj from Kubernetes cluster and records prometheus metrics.
func (c *Client) Delete(ctx context.Context, obj runtime.Object, opts ...client.DeleteOptionFunc) error {
	description, kind := resourceInfo(obj)
	operation := string(action.DeleteOperation)
	action.Actions.WithLabelValues(kind, operation).Inc()
	action.APICalls.WithLabelValues(kind, operation).Inc()
	timer := prometheus.NewTimer(action.APICallDuration.WithLabelValues(kind, operation))
	defer timer.ObserveDuration()
	_, namespacedName := metaNamespacedName(obj)
	if err := c.Client.Get(ctx, namespacedName, obj); err != nil {
		if apierrors.IsNotFound(err) {
			// Object is already deleted
			return nil
		}
		if action.IsFinalizing(obj.(metav1.Object)) {
			glog.V(3).Infof("Delete skipped, resource is finalizing %s", description)
			return nil
		}
		return errors.Wrapf(err, "could not look up object we're deleting %s", description)
	}
	if err := c.Client.Delete(ctx, obj, opts...); err != nil {
		if apierrors.IsNotFound(err) {
			glog.V(3).Infof("Not found during attempted delete %s", description)
			return nil
		}
		return errors.Wrapf(err, "delete failed for %s", description)
	}
	glog.V(1).Infof("Delete OK for %s", description)
	return nil
}

// Update updates the given obj in the Kubernetes cluster.
func (c *Client) Update(ctx context.Context, obj runtime.Object, updateFn action.Update) (runtime.Object, error) {
	return c.update(ctx, obj, updateFn, c.Client.Update)
}

// UpdateStatus updates the given obj's status in the Kubernetes cluster.
func (c *Client) UpdateStatus(ctx context.Context, obj runtime.Object, updateFn action.Update) (runtime.Object, error) {
	return c.update(ctx, obj, updateFn, c.Client.Status().Update)
}

// update updates the given obj in the Kubernetes cluster using clientUpdateFn and records prometheus
// metrics. In the event of a conflicting update, it will retry.
// This operation always involves retrieving the resource from API Server before actually updating it.
// Refer to action package for expected return values for updateFn.
func (c *Client) update(ctx context.Context, obj runtime.Object, updateFn action.Update,
	clientUpdateFn clientUpdateFn) (runtime.Object, error) {
	// We only want to modify the argument after successfully making an update to API Server.
	workingObj := obj.DeepCopyObject()
	description, kind := resourceInfo(workingObj)
	operation := string(action.UpdateOperation)
	action.Actions.WithLabelValues(kind, operation).Inc()
	_, namespacedName := metaNamespacedName(workingObj)
	var lastErr error
	var oldObj runtime.Object
	for tryNum := 0; tryNum < c.MaxTries; tryNum++ {
		if err := c.Client.Get(ctx, namespacedName, workingObj); err != nil {
			return nil, errors.Wrapf(err, "could not update %s; it does not exist", description)
		}
		oldV := resourceVersion(workingObj)
		newObj, err := updateFn(workingObj.DeepCopyObject())
		if err != nil {
			if action.IsNoUpdateNeeded(err) {
				return newObj, nil
			}
			return nil, err
		}

		action.APICalls.WithLabelValues(kind, operation).Inc()
		timer := prometheus.NewTimer(action.APICallDuration.WithLabelValues(kind, operation))
		if glog.V(3) {
			glog.Warningf("update: %q: try: %v diff old..new:\n%v",
				namespacedName, tryNum, cmp.Diff(workingObj, newObj))
			if oldObj != nil {
				glog.Warningf("update: %q: prev..old:\n%v",
					namespacedName, cmp.Diff(oldObj, workingObj))
			}
		}
		err = clientUpdateFn(ctx, newObj)
		timer.ObserveDuration()
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
			return nil, err
		}
		<-time.After(100 * time.Millisecond) // Back off on retry a bit.
	}
	return nil, errors.Errorf("%v tries exceeded for %s, last error: %v", c.MaxTries, description, lastErr)
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

	operation := string(action.UpdateOperation)
	action.Actions.WithLabelValues(kind, operation).Inc()
	action.APICalls.WithLabelValues(kind, operation).Inc()
	timer := prometheus.NewTimer(action.APICallDuration.WithLabelValues(kind, operation))
	defer timer.ObserveDuration()
	if err := c.Client.Update(ctx, obj); err != nil {
		return errors.Wrapf(err, "upsert failed for %s", description)
	}
	glog.V(1).Infof("Upsert OK for %s", description)
	return nil
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
