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
	fLogger          fightLogger
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
		fights:           newFightDetector(),
		fLogger:          newFightLogger(),
	}, nil
}

// Create implements Applier.
func (c *clientApplier) Create(ctx context.Context, intendedState *unstructured.Unstructured) (bool, status.Error) {
	err := c.create(ctx, intendedState)
	metrics.Operations.WithLabelValues("create", intendedState.GetKind(), metrics.StatusLabel(err)).Inc()

	if err != nil {
		return false, status.ResourceWrap(err, "unable to create resource", ast.ParseFileObject(intendedState))
	}
	if fight := c.fights.markUpdated(time.Now(), ast.NewFileObject(intendedState, cmpath.RelativeSlash(""))); fight != nil {
		if c.fLogger.logFight(time.Now(), fight) {
			glog.Warningf("Fight detected on create of %s.", description(intendedState))
		}
	}
	return true, nil
}

// Update implements Applier.
func (c *clientApplier) Update(ctx context.Context, intendedState, currentState *unstructured.Unstructured) (bool, status.Error) {
	patch, err := c.update(ctx, intendedState, currentState)
	metrics.Operations.WithLabelValues("update", intendedState.GetKind(), metrics.StatusLabel(err)).Inc()
	if err != nil {
		return false, status.ResourceWrap(err, "unable to update resource", ast.ParseFileObject(intendedState))
	}

	updated := patch != nil && !isNoOpPatch(patch)
	if updated {
		if fight := c.fights.markUpdated(time.Now(), ast.NewFileObject(intendedState, cmpath.RelativeSlash(""))); fight != nil {
			if c.fLogger.logFight(time.Now(), fight) {
				glog.Warningf("Fight detected on update of %s which applied the following patch:\n%s", description(intendedState), string(patch))
			}
		}
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
	if fight := c.fights.markUpdated(time.Now(), ast.NewFileObject(obj, cmpath.RelativeSlash(""))); fight != nil {
		if c.fLogger.logFight(time.Now(), fight) {
			glog.Warningf("Fight detected on delete of %s.", description(obj))
		}
	}
	return true, nil
}

// create creates the resource with the last-applied annotation set.
func (c *clientApplier) create(ctx context.Context, obj *unstructured.Unstructured) error {
	if err := util.CreateApplyAnnotation(obj, unstructured.UnstructuredJSONScheme); err != nil {
		return errors.Wrapf(err, "could not generate apply annotation for %s", description(obj))
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
func (c *clientApplier) update(ctx context.Context, intendedState, currentState *unstructured.Unstructured) ([]byte, error) {
	// Serialize the current configuration of the object.
	current, cErr := runtime.Encode(unstructured.UnstructuredJSONScheme, currentState)
	if cErr != nil {
		return nil, errors.Errorf("could not serialize current configuration from %v", currentState)
	}

	// Retrieve the last applied configuration of the object from the annotation.
	previous, oErr := util.GetOriginalConfiguration(currentState)
	if oErr != nil {
		return nil, errors.Errorf("could not retrieve original configuration from %v", currentState)
	}

	// Serialize the modified configuration of the object, populating the last applied annotation as well.
	modified, mErr := util.GetModifiedConfiguration(intendedState, true, unstructured.UnstructuredJSONScheme)
	if mErr != nil {
		return nil, errors.Errorf("could not serialize intended configuration from %v", intendedState)
	}

	resourceClient, rErr := c.clientFor(intendedState)
	if rErr != nil {
		return nil, nil
	}

	name := intendedState.GetName()
	resourceDescription := description(intendedState)
	gvk := intendedState.GroupVersionKind()
	//TODO(b/b152322972): Add unit tests for patch return value.
	var err error

	// Attempt a strategic patch first.
	patch := c.calculateStrategic(resourceDescription, gvk, previous, modified, current)
	// If patch is nil, it means we don't have access to the schema.
	if patch != nil {
		err = attemptPatch(ctx, resourceClient, resourceDescription, name, types.StrategicMergePatchType, patch, gvk)
		// UnsupportedMediaType error indicates an invalid strategic merge patch (always true for a
		// custom resource), so we reset the patch and try again below.
		if err != nil {
			if apierrors.IsUnsupportedMediaType(err) {
				patch = nil
			} else {
				glog.Warningf("strategic merge patch for %s failed: %v", resourceDescription, err)
			}
		}
	}

	// If we weren't able to do a Strategic Merge, we fall back to JSON Merge Patch.
	if patch == nil {
		patch, err = c.calculateJSONMerge(gvk, previous, modified, current)
		if err == nil {
			err = attemptPatch(ctx, resourceClient, resourceDescription, name, types.MergePatchType, patch, gvk)
		}
	}

	if err != nil {
		return nil, errors.Wrapf(err, "could not patch resource %s with %s", resourceDescription, patch)
	}
	glog.Infof("Patched %s", resourceDescription)
	glog.V(1).Infof("Patched with %s", patch)
	return patch, nil
}

func (c *clientApplier) calculateStrategic(resourceDescription string, gvk schema.GroupVersionKind, previous, modified, current []byte) []byte {
	// Try to use schema from OpenAPI Spec if possible.
	gvkSchema := c.openAPIResources.LookupResource(gvk)
	if gvkSchema == nil {
		return nil
	}
	patchMeta := strategicpatch.PatchMetaFromOpenAPI{Schema: gvkSchema}
	patch, err := strategicpatch.CreateThreeWayMergePatch(previous, modified, current, patchMeta, true)
	if err != nil {
		glog.Infof("could not calculate a patch for %s from OpenAPI spec: %v", resourceDescription, err)
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

// isNoOpPatch returns true if the given patch is a no-op that should be ignored.
// TODO(b/152312521): Find a more elegant solution for ignoring noop-like patches.
func isNoOpPatch(patch []byte) bool {
	p := string(patch)
	return p == noOpPatch || p == rmCreationTimestampPatch
}

// attemptPatch patches resClient with the given patch.
// TODO(b/152322972): Add unit tests for noop logic.
func attemptPatch(ctx context.Context, resClient dynamic.ResourceInterface, resourceDescription string, name string, patchType types.PatchType, patch []byte, gvk schema.GroupVersionKind) error {
	if isNoOpPatch(patch) {
		glog.V(3).Infof("Ignoring no-op patch %s for %q", patch, resourceDescription)
		return nil
	}

	if err := ctx.Err(); err != nil {
		// We've already encountered an error, so do not attempt update.
		return errors.Wrapf(err, "patch cancelled due to context error")
	}

	start := time.Now()
	_, err := resClient.Patch(name, patchType, patch, metav1.UpdateOptions{})
	duration := time.Since(start).Seconds()
	metrics.APICallDuration.WithLabelValues("patch", gvk.String(), metrics.StatusLabel(err)).Observe(duration)
	return err
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

	return "", errors.Errorf("could not find plural resource name for %s", gvk)
}

func description(u *unstructured.Unstructured) string {
	name := u.GetName()
	namespace := u.GetNamespace()
	gvk := u.GroupVersionKind()
	if namespace == "" {
		return fmt.Sprintf("[%s kind=%s name=%s]", gvk.GroupVersion(), gvk.Kind, name)
	}
	return fmt.Sprintf("[%s kind=%s namespace=%s name=%s]", gvk.GroupVersion(), gvk.Kind, namespace, name)
}
