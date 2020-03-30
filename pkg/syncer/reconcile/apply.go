package reconcile

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/metrics"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
	"k8s.io/kubectl/pkg/util"
	"k8s.io/kubectl/pkg/util/openapi"
)

const noOpPatch = "{}"
const rmCreationTimestampPatch = "{\"metadata\":{\"creationTimestamp\":null}}"

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
	fights           fightDetector
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
	c.fights.markUpdated(time.Now(), ast.NewFileObject(intendedState, cmpath.FromOS("")))
	return true, nil
}

// Update implements Applier.
func (c *clientApplier) Update(ctx context.Context, intendedState, currentState *unstructured.Unstructured) (bool, status.Error) {
	updated, err := c.update(ctx, intendedState, currentState)
	metrics.Operations.WithLabelValues("update", intendedState.GetKind(), metrics.StatusLabel(err)).Inc()

	if err != nil {
		return false, status.ResourceWrap(err, "unable to update resource", ast.ParseFileObject(intendedState))
	}
	if updated {
		c.fights.markUpdated(time.Now(), ast.NewFileObject(intendedState, cmpath.FromOS("")))
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
	c.fights.markUpdated(time.Now(), ast.NewFileObject(obj, cmpath.FromOS("")))
	return true, nil
}

// create creates the resource with the last-applied annotation set.
func (c *clientApplier) create(ctx context.Context, obj *unstructured.Unstructured) error {
	if err := util.CreateApplyAnnotation(obj, unstructured.UnstructuredJSONScheme); err != nil {
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
// The implementation here has been mostly extracted from the apply command: k8s.io/kubectl/cmd/apply.go
func (c *clientApplier) update(ctx context.Context, intendedState, currentState *unstructured.Unstructured) (bool, error) {
	// Serialize the current configuration of the object.
	current, cErr := runtime.Encode(unstructured.UnstructuredJSONScheme, currentState)
	if cErr != nil {
		return false, errors.Errorf("could not serialize current configuration from %v", currentState)
	}

	// Retrieve the last applied configuration of the object from the annotation.
	previous, oErr := util.GetOriginalConfiguration(currentState)
	if oErr != nil {
		return false, errors.Errorf("could not retrieve original configuration from %v", currentState)
	}

	// Serialize the modified configuration of the object, populating the last applied annotation as well.
	modified, mErr := util.GetModifiedConfiguration(intendedState, true, unstructured.UnstructuredJSONScheme)
	if mErr != nil {
		return false, errors.Errorf("could not serialize intended configuration from %v", intendedState)
	}

	resourceClient, rErr := c.clientFor(intendedState)
	if rErr != nil {
		return false, nil
	}

	name, resourceDescription := nameDescription(intendedState)
	gvk := intendedState.GroupVersionKind()
	//TODO(b/b152322972): Add unit tests for "updated" logic.
	var updated bool
	var err error

	// Attempt a strategic patch first.
	patch := c.calculateStrategic(gvk, previous, modified, current)
	// If patch is nil, it means we don't have access to the schema.
	if patch != nil {
		updated, err = attemptPatch(ctx, resourceClient, name, types.StrategicMergePatchType, patch, gvk)
		// UnsupportedMediaType error indicates an invalid strategic merge patch (always true for a
		// custom resource), so we reset the patch and try again below.
		if err != nil && apierrors.IsUnsupportedMediaType(err) {
			patch = nil
		}
	}

	// If we weren't able to do a Strategic Merge, we fall back to JSON Merge Patch.
	if patch == nil {
		patch, err = c.calculateJSONMerge(gvk, previous, modified, current)
		if err == nil {
			updated, err = attemptPatch(ctx, resourceClient, name, types.MergePatchType, patch, gvk)
		}
	}

	if err != nil {
		return false, errors.Wrap(err, "could not patch")
	}
	glog.V(1).Infof("Patched %s", resourceDescription)
	glog.V(3).Infof("Patched with %s", patch)

	return updated, nil
}

func (c *clientApplier) calculateStrategic(gvk schema.GroupVersionKind, previous, modified, current []byte) []byte {
	// Try to use schema from OpenAPI Spec if possible.
	gvkSchema := c.openAPIResources.LookupResource(gvk)
	if gvkSchema == nil {
		return nil
	}
	patchMeta := strategicpatch.PatchMetaFromOpenAPI{Schema: gvkSchema}
	patch, err := strategicpatch.CreateThreeWayMergePatch(previous, modified, current, patchMeta, true)
	if err != nil {
		glog.Warning(errors.Wrap(err, "could not calculate the patch from OpenAPI spec"))
		return nil
	}
	return patch
}

func (c *clientApplier) calculateJSONMerge(gvk schema.GroupVersionKind, previous, modified, current []byte) ([]byte, error) {
	preconditions := []mergepatch.PreconditionFunc{
		mergepatch.RequireKeyUnchanged("apiVersion"),
		mergepatch.RequireKeyUnchanged("kind"),
		mergepatch.RequireMetadataKeyUnchanged("name"),
	}
	return jsonmergepatch.CreateThreeWayJSONMergePatch(previous, modified, current, preconditions...)
}

// attemptPatch patches resClient with the given patch.
//
// Returns (true, nil) if the object was successfully patched.
// Returns (false, nil) if the patch was a no-op.
// Returns (false, err) if the patch failed.
// TODO(b/152322972): Add unit tests for noop logic.
func attemptPatch(ctx context.Context, resClient dynamic.ResourceInterface, name string, patchType types.PatchType, patch []byte, gvk schema.GroupVersionKind) (bool, error) {
	if p := string(patch); p == noOpPatch || p == rmCreationTimestampPatch {
		// TODO(b/152312521): Find a more elegant solution for ignoring noop-like patches.
		// Avoid doing a noop patch.
		return false, nil
	}

	if err := ctx.Err(); err != nil {
		// We've already encountered an error, so do not attempt update.
		return false, status.ResourceWrap(err, "unable to continue updating resource")
	}

	start := time.Now()
	_, err := resClient.Patch(name, patchType, patch, metav1.UpdateOptions{})
	duration := time.Since(start).Seconds()
	metrics.APICallDuration.WithLabelValues("patch", gvk.String(), metrics.StatusLabel(err)).Observe(duration)
	return true, err
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
