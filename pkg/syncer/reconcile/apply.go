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
	"k8s.io/kubernetes/pkg/kubectl/cmd/util/openapi"
	"k8s.io/kubernetes/pkg/kubectl/scheme"
)

// Applier updates a resource from its current state to its intended state using apply operations.
type Applier interface {
	Create(ctx context.Context, obj *unstructured.Unstructured) (bool, status.Error)
	Update(ctx context.Context, intendedState, currentState *unstructured.Unstructured) (bool, status.Error)
	Delete(ctx context.Context, obj *unstructured.Unstructured) (bool, status.Error)
}

// ClientApplier does apply operations on resources, client-side, using the same approach as running `kubectl apply`.
type ClientApplier struct {
	dynamicClient    dynamic.Interface
	discoveryClient  *discovery.DiscoveryClient
	openAPIResources openapi.Resources
	client           *client.Client
}

var _ Applier = &ClientApplier{}

// NewApplier returns a new ClientApplier.
func NewApplier(cfg *rest.Config, client *client.Client) (*ClientApplier, error) {
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

	return &ClientApplier{
		dynamicClient:    c,
		discoveryClient:  dc,
		openAPIResources: oa,
		client:           client,
	}, nil
}

// Create implements Applier.
func (c *ClientApplier) Create(ctx context.Context, intendedState *unstructured.Unstructured) (bool, status.Error) {
	err := c.create(ctx, intendedState)
	metrics.Operations.WithLabelValues("create", intendedState.GetKind(), metrics.StatusLabel(err)).Inc()

	if err != nil {
		return false, status.ResourceWrap(err, "unable to create resource", ast.ParseFileObject(intendedState))
	}
	return true, nil
}

// Update implements Applier.
func (c *ClientApplier) Update(ctx context.Context, intendedState, currentState *unstructured.Unstructured) (bool, status.Error) {
	updated, err := c.update(ctx, intendedState, currentState)
	metrics.Operations.WithLabelValues("update", intendedState.GetKind(), metrics.StatusLabel(err)).Inc()

	if err != nil {
		return false, status.ResourceWrap(err, "unable to update resource", ast.ParseFileObject(intendedState))
	}
	return updated, nil
}

// Delete implements Applier.
func (c *ClientApplier) Delete(ctx context.Context, obj *unstructured.Unstructured) (bool, status.Error) {
	err := c.client.Delete(ctx, obj)
	metrics.Operations.WithLabelValues("delete", obj.GetKind(), metrics.StatusLabel(err)).Inc()

	if err != nil {
		return false, status.ResourceWrap(err, "unable to delete resource", ast.ParseFileObject(obj))
	}
	return true, nil
}

// create creates the resource with the last-applied annotation set.
func (c *ClientApplier) create(ctx context.Context, obj *unstructured.Unstructured) error {
	if err := kubectl.CreateApplyAnnotation(obj, unstructured.UnstructuredJSONScheme); err != nil {
		return errors.Wrap(err, "could not generate apply annotation")
	}

	return c.client.Create(ctx, obj)
}

// clientFor returns the client which may interact with the passed object.
func (c *ClientApplier) clientFor(obj *unstructured.Unstructured) (dynamic.ResourceInterface, error) {
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
func (c *ClientApplier) update(ctx context.Context, intendedState, currentState *unstructured.Unstructured) (bool, error) {
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

	var patch []byte
	var patchType types.PatchType

	gvk := intendedState.GetObjectKind().GroupVersionKind()
	versionedObject, sErr := scheme.Scheme.New(gvk)
	_, unversioned := scheme.Scheme.IsUnversioned(intendedState)
	switch {
	case runtime.IsNotRegisteredError(sErr) || unversioned:
		preconditions := []mergepatch.PreconditionFunc{
			mergepatch.RequireKeyUnchanged("apiVersion"),
			mergepatch.RequireKeyUnchanged("kind"),
			mergepatch.RequireMetadataKeyUnchanged("name"),
		}
		var err error
		patch, err = jsonmergepatch.CreateThreeWayJSONMergePatch(previous, modified, current, preconditions...)
		patchType = types.MergePatchType
		if err != nil {
			if mergepatch.IsPreconditionFailed(err) {
				return false, errors.New("at least one of apiVersion, kind and name was changed")
			}
			return false, errors.Wrap(err, "could not calculate the patch")
		}
	case sErr != nil:
		return false, errors.Wrapf(sErr, "could not get an instance of versioned object %s", gvk)
	case sErr == nil:
		// Compute a three way strategic merge patch to send to server.
		patchType = types.StrategicMergePatchType

		if s := c.openAPIResources.LookupResource(gvk); s != nil {
			// Try to use openapi first if the openapi spec is available and can successfully calculate the patch.
			// Otherwise, fall back to baked-in types for creating the patch.
			lookupPatchMeta := strategicpatch.PatchMetaFromOpenAPI{Schema: s}
			openAPIPatch, err := strategicpatch.CreateThreeWayMergePatch(previous, modified, current, lookupPatchMeta, true)
			if err != nil {
				glog.Warning(errors.Wrap(err, "could not calculate the patch from openapi spec"))
			} else {
				patch = openAPIPatch
			}
		}

		if patch == nil {
			lookupPatchMeta, err := strategicpatch.NewPatchMetaFromStruct(versionedObject)
			if err != nil {
				return false, err
			}
			patch, err = strategicpatch.CreateThreeWayMergePatch(previous, modified, current, lookupPatchMeta, true)
			if err != nil {
				return false, err
			}
		}
	}

	if string(patch) == "{}" {
		// Avoid doing a noop patch.
		return false, nil
	}

	if ctx.Err() != nil {
		// We've already encountered an error, so do not attempt update.
		return false, status.ResourceWrap(ctx.Err(), "unable to continue updating resources")
	}

	name, resourceDescription := nameDescription(intendedState)
	start := time.Now()
	_, err := resourceClient.Patch(name, patchType, patch, metav1.UpdateOptions{})
	duration := time.Since(start).Seconds()
	metrics.APICallDuration.WithLabelValues("patch", gvk.String(), metrics.StatusLabel(err)).Observe(duration)

	if err != nil {
		return false, errors.Wrap(err, "could not patch")
	}
	glog.V(1).Infof("Patched %s", resourceDescription)
	glog.V(3).Infof("Patched with %s", patch)

	return true, nil
}

// resource retrieves the plural resource name for the GroupVersionKind.
func (c *ClientApplier) resource(gvk schema.GroupVersionKind) (string, error) {
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
	return name, fmt.Sprintf("%s, %s", u.GetObjectKind().GroupVersionKind(), name)
}
