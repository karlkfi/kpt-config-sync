package reconcile

import (
	"context"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
	syncerclient "github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/metrics"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/jsonmergepatch"
	"k8s.io/apimachinery/pkg/util/mergepatch"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/kubectl/pkg/util"
	"k8s.io/kubectl/pkg/util/openapi"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	openAPIResources openapi.Resources
	client           *syncerclient.Client
	fights           fightDetector
	fLogger          fightLogger
}

var _ Applier = &clientApplier{}

// NewApplier returns a new clientApplier.
func NewApplier(oa openapi.Resources, client *syncerclient.Client) Applier {
	return &clientApplier{
		openAPIResources: oa,
		client:           client,
		fights:           newFightDetector(),
		fLogger:          newFightLogger(),
	}
}

// Create implements Applier.
func (c *clientApplier) Create(ctx context.Context, intendedState *unstructured.Unstructured) (bool, status.Error) {
	err := c.create(ctx, intendedState)
	metrics.Operations.WithLabelValues("create", intendedState.GetKind(), metrics.StatusLabel(err)).Inc()

	if err != nil {
		return false, status.ResourceWrap(err, "unable to create resource", ast.ParseFileObject(intendedState))
	}
	if fight := c.fights.markUpdated(time.Now(), ast.NewFileObject(intendedState, cmpath.FromOS(""))); fight != nil {
		if c.fLogger.logFight(time.Now(), fight) {
			glog.Warningf("Fight detected on create of %s/%s.", intendedState.GetNamespace(), intendedState.GetName())
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
		if fight := c.fights.markUpdated(time.Now(), ast.NewFileObject(intendedState, cmpath.FromOS(""))); fight != nil {
			if c.fLogger.logFight(time.Now(), fight) {
				glog.Warningf("Fight detected on update of %s/%s which applied the following patch:\n%s", intendedState.GetNamespace(), intendedState.GetName(), string(patch))
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
	if fight := c.fights.markUpdated(time.Now(), ast.NewFileObject(obj, cmpath.FromOS(""))); fight != nil {
		if c.fLogger.logFight(time.Now(), fight) {
			glog.Warningf("Fight detected on delete of %s/%s.", obj.GetNamespace(), obj.GetName())
		}
	}
	return true, nil
}

// create creates the resource with the last-applied annotation set.
func (c *clientApplier) create(ctx context.Context, obj *unstructured.Unstructured) error {
	if err := util.CreateApplyAnnotation(obj, unstructured.UnstructuredJSONScheme); err != nil {
		return errors.Wrap(err, "could not generate apply annotation")
	}

	return c.client.Create(ctx, obj)
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

	gvk := intendedState.GroupVersionKind()
	//TODO(b/b152322972): Add unit tests for patch return value.

	// Attempt a strategic patch first.
	patch := c.calculateStrategic(gvk, previous, modified, current)
	// If patch is nil, it means we don't have access to the schema.
	if patch != nil {
		err := attemptPatch(ctx, c.client.Client, intendedState, types.StrategicMergePatchType, patch)
		// UnsupportedMediaType error indicates an invalid strategic merge patch (always true for a
		// custom resource), so we reset the patch and try again below.
		if err != nil && apierrors.IsUnsupportedMediaType(err) {
			patch = nil
		}
	}

	var err error
	// If we weren't able to do a Strategic Merge, we fall back to JSON Merge Patch.
	if patch == nil {
		patch, err = c.calculateJSONMerge(gvk, previous, modified, current)
		if err == nil {
			err = attemptPatch(ctx, c.client.Client, intendedState, types.MergePatchType, patch)
		}
	}

	if err != nil {
		return nil, errors.Wrap(err, "could not patch")
	}
	resourceDescription := core.IDOf(intendedState).String()
	glog.V(1).Infof("Patched %s", resourceDescription)
	glog.V(3).Infof("Patched with %s", patch)

	return patch, nil
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

// isNoOpPatch returns true if the given patch is a no-op that should be ignored.
// TODO(b/152312521): Find a more elegant solution for ignoring noop-like patches.
func isNoOpPatch(patch []byte) bool {
	p := string(patch)
	return p == noOpPatch || p == rmCreationTimestampPatch
}

// attemptPatch patches resClient with the given patch.
//
// obj is only used to identify the resource to patch. The actual update
// information is fully contained in `patch`.
//
// TODO(b/152322972): Add unit tests for noop logic.
func attemptPatch(ctx context.Context, c client.Writer, intended core.Object, patchType types.PatchType, patch []byte) error {
	if isNoOpPatch(patch) {
		glog.V(3).Infof("Ignoring no-op patch for %q", intended.GetName())
		return nil
	}

	if err := ctx.Err(); err != nil {
		// We've already encountered an error, so do not attempt update.
		return status.ResourceWrap(err, "unable to continue updating resource")
	}

	start := time.Now()
	err := c.Patch(ctx, intended, client.ConstantPatch(patchType, patch))
	duration := time.Since(start).Seconds()
	metrics.APICallDuration.WithLabelValues("patch", intended.GroupVersionKind().String(), metrics.StatusLabel(err)).Observe(duration)
	return err
}
