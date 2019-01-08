package reconcile

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/jsonmergepatch"
	"k8s.io/apimachinery/pkg/util/mergepatch"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/kubectl"
	"k8s.io/kubernetes/pkg/kubectl/cmd/util/openapi"
)

// Applier updates a resource from its current state to its intended state using apply operations.
type Applier interface {
	ApplyCluster(intendedState, currentState runtime.Object) error
	ApplyNamespace(namespace string, intendedState, currentState runtime.Object) error
}

// ClientApplier does apply operations on resources, client-side, using the same approach as running `kubectl apply`.
type ClientApplier struct {
	dynamicClient    dynamic.Interface
	discoveryClient  *discovery.DiscoveryClient
	openAPIResources openapi.Resources
}

// NewApplier returns a new ClientApplier.
func NewApplier(cfg *rest.Config) (*ClientApplier, error) {
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
	}, nil
}

// ApplyCluster applies a patch to the cluster-scoped resource to move from currentState to intendedState.
func (c *ClientApplier) ApplyCluster(intendedState, currentState runtime.Object) error {
	return c.apply("", false, intendedState, currentState)
}

// ApplyNamespace applies a patch to the namespace-scoped resource to move from currentState to intendedState.
func (c *ClientApplier) ApplyNamespace(namespace string, intendedState, currentState runtime.Object) error {
	return c.apply(namespace, true, intendedState, currentState)
}

// apply updates a resource using the same approach as running `kubectl apply`.
// The implementation here has been mostly extracted from the apply command: k8s.io/kubernetes/pkg/kubectl/cmd/apply.go
func (c *ClientApplier) apply(namespace string, namespaceable bool, intendedState, currentState runtime.Object) error {
	// Serialize the current configuration of the object.
	current, cErr := runtime.Encode(unstructured.UnstructuredJSONScheme, currentState)
	if cErr != nil {
		return errors.Errorf("could not serialize current configuration from %v", currentState)
	}

	// Retrieve the last applied configuration of the object from the annotation.
	previous, oErr := kubectl.GetOriginalConfiguration(currentState)
	if oErr != nil {
		return errors.Errorf("could not retrieve original configuration from %v", currentState)
	}

	// Serialize the modified configuration of the object, populating the last applied annotation as well.
	modified, mErr := kubectl.GetModifiedConfiguration(intendedState, true, unstructured.UnstructuredJSONScheme)
	if mErr != nil {
		return errors.Errorf("could not serialize current configuration from %v", intendedState)
	}

	gvk := currentState.GetObjectKind().GroupVersionKind()
	resource, rErr := c.resource(gvk)
	if rErr != nil {
		return rErr
	}

	gvr := gvk.GroupVersion().WithResource(resource)
	var client dynamic.ResourceInterface
	if namespaceable {
		client = c.dynamicClient.Resource(gvr).Namespace(namespace)
	} else {
		client = c.dynamicClient.Resource(gvr)
	}

	name, err := meta.NewAccessor().Name(intendedState)
	if err != nil {
		return errors.Wrapf(err, "could not extract name from %s", intendedState)
	}
	resourceDescription := fmt.Sprintf("%s, %s", gvk, name)

	var patch []byte
	var patchType types.PatchType

	versionedObject, sErr := scheme.Scheme.New(gvk)
	switch {
	case runtime.IsNotRegisteredError(sErr):
		preconditions := []mergepatch.PreconditionFunc{
			mergepatch.RequireKeyUnchanged("apiVersion"),
			mergepatch.RequireKeyUnchanged("kind"),
			mergepatch.RequireMetadataKeyUnchanged("name"),
		}
		patch, err = jsonmergepatch.CreateThreeWayJSONMergePatch(previous, modified, current, preconditions...)
		patchType = types.MergePatchType
		if err != nil {
			if mergepatch.IsPreconditionFailed(err) {
				return errors.Errorf("At least one of apiVersion, kind and name was changed for %s", resourceDescription)
			}
			return errors.Wrapf(err, "could not calculate the patch for %s", resourceDescription)
		}
	case sErr != nil:
		return errors.Wrapf(sErr, "could not get an instance of versioned object %s", gvk)
	case sErr == nil:
		// Compute a three way strategic merge patch to send to server.
		patchType = types.StrategicMergePatchType

		if s := c.openAPIResources.LookupResource(gvk); s != nil {
			// Try to use openapi first if the openapi spec is available and can successfully calculate the patch.
			// Otherwise, fall back to baked-in types for creating the patch.
			lookupPatchMeta := strategicpatch.PatchMetaFromOpenAPI{Schema: s}
			openAPIPatch, err := strategicpatch.CreateThreeWayMergePatch(previous, modified, current, lookupPatchMeta, true)
			if err != nil {
				glog.Warning(errors.Wrapf(err, "could not calculate the patch from openapi spec for %s", resourceDescription))
			} else {
				patch = openAPIPatch
			}
		}

		if patch == nil {
			lookupPatchMeta, err := strategicpatch.NewPatchMetaFromStruct(versionedObject)
			if err != nil {
				return err
			}
			patch, err = strategicpatch.CreateThreeWayMergePatch(previous, modified, current, lookupPatchMeta, true)
			if err != nil {
				return err
			}
		}
	}

	if string(patch) == "{}" {
		// Avoid doing a noop patch.
		return nil
	}

	if _, err := client.Patch(name, patchType, patch); err != nil {
		return errors.Wrapf(err, "could not patch %s", resourceDescription)
	}
	glog.V(3).Infof("Patching %s with %s", resourceDescription, patch)

	return nil
}

// resource retrieves the plural resource name for the GroupVersionKind.
func (c *ClientApplier) resource(gvk schema.GroupVersionKind) (string, error) {
	apiResources, err := c.discoveryClient.ServerResourcesForGroupVersion(gvk.GroupVersion().String())
	if err != nil {
		return "", err
	}

	for _, r := range apiResources.APIResources {
		if r.Kind == gvk.Kind {
			return r.Name, nil
		}
	}

	return "", errors.Errorf("could not find resource for %s", gvk)
}
