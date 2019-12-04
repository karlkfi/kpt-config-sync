package reconcile

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/metrics"
	"github.com/google/nomos/third_party/k8s.io/kubernetes/pkg/kubectl"
	"github.com/google/nomos/third_party/k8s.io/kubernetes/pkg/kubectl/cmd/util/openapi"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/jsonmergepatch"
	"k8s.io/apimachinery/pkg/util/mergepatch"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

// Applier updates a resource from its current state to its intended state using apply operations.
type Applier interface {
	Create(ctx context.Context, obj *unstructured.Unstructured) (bool, status.Error)
	Update(ctx context.Context, intendedState, currentState *unstructured.Unstructured) (bool, status.Error)
	Delete(ctx context.Context, obj *unstructured.Unstructured) (bool, status.Error)
}

// clientApplier does apply operations on resources, client-side, using the same approach as running `kubectl apply`.
type clientApplier struct {
	dynamicClient    dynamic.Interface
	discoveryClient  *discovery.DiscoveryClient
	openAPIResources openapi.Resources
	client           *client.Client
}

var _ Applier = &clientApplier{}

// NewApplier returns a new clientApplier.
func NewApplier(cfg *rest.Config, client *client.Client) (Applier, error) {
	c, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, err
	}

	oa, err := openapi.NewOpenAPIGetter(dc).Get()
	if err != nil {
		return nil, err
	}

	return &clientApplier{
		dynamicClient:    c,
		discoveryClient:  dc,
		openAPIResources: oa,
		client:           client,
	}, nil
}

// Create implements Applier.
func (c *clientApplier) Create(ctx context.Context, intendedState *unstructured.Unstructured) (bool, status.Error) {
	err := c.create(ctx, intendedState)
	metrics.Operations.WithLabelValues("create", intendedState.GetKind(), metrics.StatusLabel(err)).Inc()

	if err != nil {
		return false, status.ResourceWrap(err, "unable to create resource", ast.ParseFileObject(intendedState))
	}
	return true, nil
}

// Update implements Applier.
func (c *clientApplier) Update(ctx context.Context, intendedState, currentState *unstructured.Unstructured) (bool, status.Error) {
	updated, err := c.update(ctx, intendedState, currentState)
	metrics.Operations.WithLabelValues("update", intendedState.GetKind(), metrics.StatusLabel(err)).Inc()

	if err != nil {
		return false, status.ResourceWrap(err, "unable to update resource", ast.ParseFileObject(intendedState))
	}
	return updated, nil
}

// Delete implements Applier.
func (c *clientApplier) Delete(ctx context.Context, obj *unstructured.Unstructured) (bool, status.Error) {
	err := c.client.Delete(ctx, obj)
	metrics.Operations.WithLabelValues("delete", obj.GetKind(), metrics.StatusLabel(err)).Inc()

	if err != nil {
		return false, status.ResourceWrap(err, "unable to delete resource", ast.ParseFileObject(obj))
	}
	return true, nil
}

// create creates the resource with the last-applied annotation set.
func (c *clientApplier) create(ctx context.Context, obj *unstructured.Unstructured) error {
	if err := kubectl.CreateApplyAnnotation(obj, unstructured.UnstructuredJSONScheme); err != nil {
		return errors.Wrap(err, "could not generate apply annotation")
	}

	return c.client.Create(ctx, obj)
}

// clientFor returns the client which may interact with the passed object.
func (c *clientApplier) clientFor(obj *unstructured.Unstructured) (dynamic.ResourceInterface, error) {
	gvk := obj.GroupVersionKind()
	apiResource, rErr := c.resource(gvk)
	if rErr != nil {
		return nil, errors.Wrapf(rErr, "unable to get resource client for %q", gvk.String())
	}

	gvr := gvk.GroupVersion().WithResource(apiResource)
	// If namespace is the empty string (as is the case for cluster-scoped resources), the
	// client correctly returns itself.
	return c.dynamicClient.Resource(gvr).Namespace(obj.GetNamespace()), nil
}

// apply updates a resource using the same approach as running `kubectl apply`.
// The implementation here has been mostly extracted from the apply command: k8s.io/kubernetes/pkg/kubectl/cmd/apply.go
func (c *clientApplier) update(ctx context.Context, intendedState, currentState *unstructured.Unstructured) (bool, error) {
	// Serialize the current configuration of the object.
	current, cErr := runtime.Encode(unstructured.UnstructuredJSONScheme, currentState)
	if cErr != nil {
		return false, errors.Errorf("could not serialize current configuration from %v", currentState)
	}

	// Retrieve the last applied configuration of the object from the annotation.
	previous, oErr := kubectl.GetOriginalConfiguration(currentState)
	if oErr != nil {
		return false, errors.Errorf("could not retrieve original configuration from %v", currentState)
	}

	// Serialize the modified configuration of the object, populating the last applied annotation as well.
	modified, mErr := kubectl.GetModifiedConfiguration(intendedState, true, unstructured.UnstructuredJSONScheme)
	if mErr != nil {
		return false, errors.Errorf("could not serialize intended configuration from %v", intendedState)
	}

	resourceClient, rErr := c.clientFor(intendedState)
	if rErr != nil {
		return false, nil
	}

	gvk := intendedState.GroupVersionKind()
	patch, patchType, err := c.calculatePatch(gvk, previous, modified, current)
	if err != nil {
		return false, err
	}

	if string(patch) == "{}" {
		// Avoid doing a noop patch.
		return false, nil
	}

	if err := ctx.Err(); err != nil {
		// We've already encountered an error, so do not attempt update.
		return false, status.ResourceWrap(err, "unable to continue updating resources")
	}

	name, resourceDescription := nameDescription(intendedState)
	start := time.Now()
	_, err = resourceClient.Patch(name, patchType, patch, metav1.PatchOptions{})
	duration := time.Since(start).Seconds()
	metrics.APICallDuration.WithLabelValues("patch", gvk.String(), metrics.StatusLabel(err)).Observe(duration)

	if err != nil {
		return false, errors.Wrap(err, "could not patch")
	}
	glog.V(1).Infof("Patched %s", resourceDescription)
	glog.V(3).Infof("Patched with %s", patch)

	return true, nil
}

// calculatePatch computes a three way strategic merge patch to send to server.
func (c *clientApplier) calculatePatch(gvk schema.GroupVersionKind, previous, modified, current []byte) ([]byte, types.PatchType, error) {
	if s := c.openAPIResources.LookupResource(gvk); s != nil {
		// Try to use schema from OpenAPI Spec. For Kubernetes 1.16 and earlier, the OpenAPI Spec does not include CRDs.
		// Starting with ~1.17, the OpenAPI Spec will include CRDs so this won't be an issue.
		patchMeta := strategicpatch.PatchMetaFromOpenAPI{Schema: s}
		patch, err := strategicpatch.CreateThreeWayMergePatch(previous, modified, current, patchMeta, true)
		if err == nil {
			return patch, types.StrategicMergePatchType, nil
		}
		glog.Warning(errors.Wrap(err, "could not calculate the patch from OpenAPI spec"))
	}

	// We weren't able to do a Strategic Merge because either:
	// 1) the strategic merge patch failed, or
	// 2) we don't have access to the schema.
	// So, fall back to JSON Merge Patch.
	preconditions := []mergepatch.PreconditionFunc{
		mergepatch.RequireKeyUnchanged("apiVersion"),
		mergepatch.RequireKeyUnchanged("kind"),
		mergepatch.RequireMetadataKeyUnchanged("name"),
	}
	patch, err := jsonmergepatch.CreateThreeWayJSONMergePatch(previous, modified, current, preconditions...)
	return patch, types.MergePatchType, err
}

// resource retrieves the plural resource name for the GroupVersionKind.
func (c *clientApplier) resource(gvk schema.GroupVersionKind) (string, error) {
	apiResources, err := c.discoveryClient.ServerResourcesForGroupVersion(gvk.GroupVersion().String())
	if err != nil {
		return "", errors.Wrapf(err, "could not look up %s using discovery API", gvk)
	}

	for _, r := range apiResources.APIResources {
		if r.Kind == gvk.Kind {
			return r.Name, nil
		}
	}

	return "", errors.Errorf("could not find resource for %s", gvk)
}

func nameDescription(u *unstructured.Unstructured) (string, string) {
	name := u.GetName()
	return name, fmt.Sprintf("%s, %s", u.GroupVersionKind(), name)
}
