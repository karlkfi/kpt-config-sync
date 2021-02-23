package reconcile

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	m "github.com/google/nomos/pkg/metrics"
	"github.com/google/nomos/pkg/status"
	syncerclient "github.com/google/nomos/pkg/syncer/client"
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
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const noOpPatch = "{}"
const rmCreationTimestampPatch = "{\"metadata\":{\"creationTimestamp\":null}}"

// Applier updates a resource from its current state to its intended state using apply operations.
type Applier interface {
	Create(ctx context.Context, obj *unstructured.Unstructured) (bool, status.Error)
	Update(ctx context.Context, intendedState, currentState *unstructured.Unstructured) (bool, status.Error)
	// RemoveNomosMeta performs a PUT (rather than a PATCH) to ensure that labels and annotations are removed.
	RemoveNomosMeta(ctx context.Context, intent *unstructured.Unstructured) (bool, status.Error)
	Delete(ctx context.Context, obj *unstructured.Unstructured) (bool, status.Error)
	GetClient() client.Client
}

// clientApplier does apply operations on resources, client-side, using the same approach as running `kubectl apply`.
type clientApplier struct {
	dynamicClient    dynamic.Interface
	discoveryClient  discovery.DiscoveryInterface
	openAPIResources openapi.Resources
	client           *syncerclient.Client
	fights           fightDetector
	fLogger          fightLogger
	multirepoEnabled bool
}

var _ Applier = &clientApplier{}

// NewApplier returns a new clientApplier.
func NewApplier(cfg *rest.Config, client *syncerclient.Client) (Applier, error) {
	return newApplier(cfg, client, false)
}

// NewApplierForMultiRepo returns a new clientApplier for callers with multi repo feature enabled.
func NewApplierForMultiRepo(cfg *rest.Config, client *syncerclient.Client) (Applier, error) {
	return newApplier(cfg, client, true)
}

func newApplier(cfg *rest.Config, client *syncerclient.Client, multirepoEnabled bool) (Applier, error) {
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
		multirepoEnabled: multirepoEnabled,
	}, nil
}

// Create implements Applier.
func (c *clientApplier) Create(ctx context.Context, intendedState *unstructured.Unstructured) (bool, status.Error) {
	var err status.Error
	// APIService is handled specially by client-side apply due to
	// https://github.com/kubernetes/kubernetes/issues/89264
	if intendedState.GroupVersionKind().GroupKind() == kinds.APIService().GroupKind() {
		err = c.create(ctx, intendedState)
	} else {
		err = c.client.Create(ctx, intendedState, client.FieldOwner(configsync.FieldManager))
	}
	metrics.Operations.WithLabelValues("create", intendedState.GetKind(), metrics.StatusLabel(err)).Inc()
	m.RecordApplyOperation(ctx, "create", m.StatusTagKey(err), intendedState.GroupVersionKind())

	if err != nil {
		return false, err
	}
	if c.fights.detectFight(ctx, time.Now(), intendedState, &c.fLogger, "create") {
		glog.Warningf("Fight detected on create of %s.", description(intendedState))
	}
	return true, nil
}

// Update implements Applier.
func (c *clientApplier) Update(ctx context.Context, intendedState, currentState *unstructured.Unstructured) (bool, status.Error) {
	patch, err := c.update(ctx, intendedState, currentState)
	metrics.Operations.WithLabelValues("update", intendedState.GetKind(), metrics.StatusLabel(err)).Inc()
	m.RecordApplyOperation(ctx, "update", m.StatusTagKey(err), intendedState.GroupVersionKind())

	switch {
	case apierrors.IsConflict(err):
		return false, syncerclient.ConflictUpdateOldVersion(err, intendedState)
	case apierrors.IsNotFound(err):
		return false, syncerclient.ConflictUpdateDoesNotExist(err, intendedState)
	case err != nil:
		return false, status.ResourceWrap(err, "unable to update resource", intendedState)
	}

	updated := !isNoOpPatch(patch)
	if updated {
		if c.fights.detectFight(ctx, time.Now(), intendedState, &c.fLogger, "update") {
			glog.Warningf("Fight detected on update of %s which applied the following patch:\n%s", description(intendedState), string(patch))
		}
	}
	return updated, nil
}

// RemoveNomosMeta implements Applier.
func (c *clientApplier) RemoveNomosMeta(ctx context.Context, u *unstructured.Unstructured) (bool, status.Error) {
	var changed bool
	_, err := c.client.Update(ctx, u, func(obj core.Object) (core.Object, error) {
		changed = RemoveNomosLabelsAndAnnotations(obj)
		if !changed {
			return obj, syncerclient.NoUpdateNeeded()
		}
		return obj, nil
	})
	metrics.Operations.WithLabelValues("update", u.GetKind(), metrics.StatusLabel(err)).Inc()
	m.RecordApplyOperation(ctx, "update", m.StatusTagKey(err), u.GroupVersionKind())

	return changed, err
}

// Delete implements Applier.
func (c *clientApplier) Delete(ctx context.Context, obj *unstructured.Unstructured) (bool, status.Error) {
	err := c.client.Delete(ctx, obj)
	metrics.Operations.WithLabelValues("delete", obj.GetKind(), metrics.StatusLabel(err)).Inc()
	m.RecordApplyOperation(ctx, "delete", m.StatusTagKey(err), obj.GroupVersionKind())

	if err != nil {
		return false, err
	}
	if c.fights.detectFight(ctx, time.Now(), obj, &c.fLogger, "delete") {
		glog.Warningf("Fight detected on delete of %s.", description(obj))
	}
	return true, nil
}

// create creates the resource with the declared-config annotation set.
func (c *clientApplier) create(ctx context.Context, obj *unstructured.Unstructured) status.Error {
	// When multi-repo feature is enabled, use kubectl last-applied-annotation.
	var err error
	if c.multirepoEnabled {
		err = util.CreateApplyAnnotation(obj, unstructured.UnstructuredJSONScheme)
	} else {
		err = createApplyAnnotation(obj, unstructured.UnstructuredJSONScheme)
	}
	if err != nil {
		return status.ResourceWrap(err, "could not generate apply annotation on create", obj)
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
	if intendedState.GroupVersionKind().GroupKind() == kinds.APIService().GroupKind() {
		return c.updateAPIService(ctx, intendedState, currentState)
	}
	objCopy := intendedState.DeepCopy()
	// Run the server-side apply dryrun first.
	// If the returned object doesn't change, skip running server-side apply.
	err := c.client.Patch(ctx, objCopy, client.Apply, client.FieldOwner(configsync.FieldManager), client.ForceOwnership, client.DryRunAll)
	if err != nil {
		return nil, err
	}
	if equal(objCopy, currentState) {
		return nil, nil
	}

	start := time.Now()
	err = c.client.Patch(ctx, intendedState, client.Apply, client.FieldOwner(configsync.FieldManager), client.ForceOwnership)
	duration := time.Since(start).Seconds()
	metrics.APICallDuration.WithLabelValues("update", intendedState.GroupVersionKind().String(), metrics.StatusLabel(err)).Observe(duration)
	m.RecordAPICallDuration(ctx, "update", m.StatusTagKey(err), intendedState.GroupVersionKind(), start)
	return []byte("updated"), err
}

// updateAPIService updates APIService type resources.
// APIService is handled specially by client-side apply due to
// https://github.com/kubernetes/kubernetes/issues/89264
func (c *clientApplier) updateAPIService(ctx context.Context, intendedState, currentState *unstructured.Unstructured) ([]byte, error) {
	resourceDescription := description(intendedState)
	// Serialize the current configuration of the object.
	current, cErr := runtime.Encode(unstructured.UnstructuredJSONScheme, currentState)
	if cErr != nil {
		return nil, errors.Errorf("could not serialize current configuration from %v", currentState)
	}

	var previous []byte
	var oErr error
	if c.multirepoEnabled {
		// Retrieve the last applied configuration of the object from the annotation.
		previous, oErr = util.GetOriginalConfiguration(currentState)
	} else {
		if err := updateConfigAnnotation(currentState); err != nil {
			return nil, errors.Wrapf(err, "could not update config annotation for %v", currentState)
		}
		// Retrieve the declared configuration of the object from the annotation.
		previous, oErr = getOriginalConfiguration(currentState)
	}
	if oErr != nil {
		return nil, errors.Errorf("could not retrieve original configuration from %v", currentState)
	}
	if previous == nil {
		if c.multirepoEnabled {
			glog.Warningf("3-way merge patch for %s may be incorrect due to missing last-applied-configuration annotation.", resourceDescription)
		} else {
			glog.Warningf("3-way merge patch for %s may be incorrect due to missing declared-config annotation.", resourceDescription)
		}
	}

	var modified []byte
	var mErr error

	if c.multirepoEnabled {
		modified, mErr = util.GetModifiedConfiguration(intendedState, true, unstructured.UnstructuredJSONScheme)
	} else {
		// Serialize the modified configuration of the object, populating the declared annotation as well.
		modified, mErr = getModifiedConfiguration(intendedState, true, unstructured.UnstructuredJSONScheme)
	}
	if mErr != nil {
		return nil, errors.Errorf("could not serialize intended configuration from %v", intendedState)
	}

	resourceClient, rErr := c.clientFor(intendedState)
	if rErr != nil {
		return nil, nil
	}

	name := intendedState.GetName()
	gvk := intendedState.GroupVersionKind()
	//TODO(b/b152322972): Add unit tests for patch return value.
	var err error

	// Attempt a strategic patch first.
	patch := c.calculateStrategic(resourceDescription, gvk, previous, modified, current)
	// If patch is nil, it means we don't have access to the schema.
	if patch != nil {
		err = attemptPatch(ctx, resourceClient, name, types.StrategicMergePatchType, patch, gvk)
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
		patch, err = c.calculateJSONMerge(previous, modified, current)
		if err == nil {
			err = attemptPatch(ctx, resourceClient, name, types.MergePatchType, patch, gvk)
		}
	}

	if err != nil {
		// Don't wrap this error. We care about it's type information, and the
		// apierrors library doesn't properly recursively check for wrapped errors.
		return nil, err
	}

	if isNoOpPatch(patch) {
		glog.V(3).Infof("Ignoring no-op patch %s for %q", patch, resourceDescription)
	} else {
		glog.Infof("Patched %s", resourceDescription)
		glog.V(1).Infof("Patched with %s", patch)
	}

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
		glog.Infof("strategic patch unavailable for %s (will use JSON patch instead): %v", resourceDescription, err)
		return nil
	}
	return patch
}

func (c *clientApplier) calculateJSONMerge(previous, modified, current []byte) ([]byte, error) {
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
	if patch == nil {
		return true
	}
	p := string(patch)
	return p == noOpPatch || p == rmCreationTimestampPatch
}

// attemptPatch patches resClient with the given patch.
// TODO(b/152322972): Add unit tests for noop logic.
func attemptPatch(ctx context.Context, resClient dynamic.ResourceInterface, name string, patchType types.PatchType, patch []byte, gvk schema.GroupVersionKind) error {
	if isNoOpPatch(patch) {
		return nil
	}

	if err := ctx.Err(); err != nil {
		// We've already encountered an error, so do not attempt update.
		return errors.Wrapf(err, "patch cancelled due to context error")
	}

	start := time.Now()
	_, err := resClient.Patch(ctx, name, patchType, patch, metav1.PatchOptions{})
	duration := time.Since(start).Seconds()
	metrics.APICallDuration.WithLabelValues("update", gvk.String(), metrics.StatusLabel(err)).Observe(duration)
	m.RecordAPICallDuration(ctx, "update", m.StatusTagKey(err), gvk, start)
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

// GetClient returns the underlying applier's client.Client.
func (c *clientApplier) GetClient() client.Client {
	return c.client.Client
}

func equal(dryrunState, currentState *unstructured.Unstructured) bool {
	cleanFields := func(u *unstructured.Unstructured) {
		u.SetGeneration(0)
		u.SetResourceVersion("")
		u.SetManagedFields(nil)
	}
	obj1 := dryrunState.DeepCopy()
	obj2 := currentState.DeepCopy()
	cleanFields(obj1)
	cleanFields(obj2)
	return reflect.DeepEqual(obj1.Object, obj2.Object)
}
