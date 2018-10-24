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

	"github.com/golang/glog"
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

// Create saves the object obj in the Kubernetes cluster and records prometheus metrics.
func (c *Client) Create(ctx context.Context, obj runtime.Object) error {
	description, resource := resourceInfo(obj)
	glog.V(1).Infof("Creating %s", description)
	operation := string(action.CreateOperation)
	action.Actions.WithLabelValues(resource, operation).Inc()
	action.APICalls.WithLabelValues(resource, operation).Inc()
	timer := prometheus.NewTimer(action.APICallDuration.WithLabelValues(resource, operation))
	defer timer.ObserveDuration()
	if err := c.Client.Create(ctx, obj); err != nil {
		return errors.Wrapf(err, "failed to create %s", description)
	}
	glog.V(1).Infof("Create OK for %s", description)
	return nil
}

// Delete deletes the given obj from Kubernetes cluster and records prometheus metrics.
func (c *Client) Delete(ctx context.Context, obj runtime.Object) error {
	description, resource := resourceInfo(obj)
	operation := string(action.DeleteOperation)
	action.Actions.WithLabelValues(resource, operation).Inc()
	action.APICalls.WithLabelValues(resource, operation).Inc()
	timer := prometheus.NewTimer(action.APICallDuration.WithLabelValues(resource, operation))
	defer timer.ObserveDuration()
	if err := c.Client.Delete(ctx, obj); err != nil {
		if apierrors.IsNotFound(err) {
			glog.V(5).Infof("not found during delete %s", description)
			return nil
		}
		return errors.Wrapf(err, "delete failed for %s", description)
	}
	glog.V(1).Infof("Delete OK for %s", description)
	return nil
}

// Update updates the given obj in the Kubernetes cluster and records prometheus metrics.
// In the event of a conflicting update, it will retry.
// This operation always involves retrieving the resource from API Server before actually updating it.
// Refer to action package for expected return values for updateFn.
func (c *Client) Update(ctx context.Context, obj runtime.Object, updateFn action.Update) (runtime.Object, error) {
	// We only want to modify the argument after successfully making an update to API Server.
	workingObj := obj.DeepCopyObject()
	description, resource := resourceInfo(workingObj)
	operation := string(action.UpdateOperation)
	action.Actions.WithLabelValues(resource, operation).Inc()
	_, namespacedName := metaNamespacedName(workingObj)
	for tryNum := 0; tryNum < c.MaxTries; tryNum++ {
		if err := c.Client.Get(ctx, namespacedName, workingObj); err != nil {
			return nil, errors.Wrapf(err, "could not update %s; it does not exist", description)
		}

		newObj, err := updateFn(workingObj)
		if err != nil {
			if action.IsNoUpdateNeeded(err) {
				return newObj, nil
			}
			return nil, err
		}

		action.APICalls.WithLabelValues(resource, operation).Inc()
		timer := prometheus.NewTimer(action.APICallDuration.WithLabelValues(resource, operation))
		err = c.Client.Update(ctx, newObj)
		timer.ObserveDuration()
		if err == nil {
			oldV, newV := resourceVersion(workingObj), resourceVersion(newObj)
			glog.V(1).Infof("Update OK for %s from ResourceVersion %s to %s", description, oldV, newV)
			return newObj, nil
		}
		if !apierrors.IsConflict(err) {
			return nil, err
		}
	}
	return nil, errors.Errorf("max tries exceeded for %s", description)
}

// Upsert creates or updates the given obj in the Kubernetes cluster and records prometheus metrics.
// This operation always involves retrieving the resource from API Server before actually creating or updating it.
func (c *Client) Upsert(ctx context.Context, obj runtime.Object) error {
	description, resource := resourceInfo(obj)
	_, namespacedName := metaNamespacedName(obj)
	if err := c.Client.Get(ctx, namespacedName, obj.DeepCopyObject()); err != nil {
		if apierrors.IsNotFound(err) {
			return c.Create(ctx, obj)
		}
		return errors.Wrapf(err, "could not get status of %s while upserting", description)
	}

	operation := string(action.UpdateOperation)
	action.Actions.WithLabelValues(resource, operation).Inc()
	action.APICalls.WithLabelValues(resource, operation).Inc()
	timer := prometheus.NewTimer(action.APICallDuration.WithLabelValues(resource, operation))
	defer timer.ObserveDuration()
	if err := c.Client.Update(ctx, obj); err != nil {
		return errors.Wrapf(err, "upsert failed for %s", description)
	}
	glog.V(1).Infof("Upsert OK for %s", description)
	return nil
}

// resourceInfo returns a description of the object (its GroupVersionKind and NamespacedName), as well as its plural name.
func resourceInfo(obj runtime.Object) (description string, resource string) {
	gvk := obj.GetObjectKind().GroupVersionKind()
	resource = action.LowerPlural(gvk.Kind)

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
