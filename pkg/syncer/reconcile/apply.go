package reconcile

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/syncer/client"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/api/core/v1"
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
	"k8s.io/kubernetes/pkg/kubectl"
	"k8s.io/kubernetes/pkg/kubectl/cmd/util/openapi"
	"k8s.io/kubernetes/pkg/kubectl/scheme"
)

// Applier updates a resource from its current state to its intended state using apply operations.
type Applier interface {
	Create(ctx context.Context, obj *unstructured.Unstructured) error
	ApplyCluster(intendedState, currentState *unstructured.Unstructured) error
	ApplyNamespace(namespace string, intendedState, currentState *unstructured.Unstructured) error
}

// ClientApplier does apply operations on resources, client-side, using the same approach as running `kubectl apply`.
// When generating the last applied annotation, we omit all the labels. This makes it so users will not inadvertently
// remove the label when `kubectl apply`ing a change that does not include the label.
type ClientApplier struct {
	dynamicClient    dynamic.Interface
	discoveryClient  *discovery.DiscoveryClient
	openAPIResources openapi.Resources
	client           *client.Client
}

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

// Create creates the resource with the last-applied annotation set.
func (c *ClientApplier) Create(ctx context.Context, obj *unstructured.Unstructured) error {
	gvk := obj.GetObjectKind().GroupVersionKind()

	if _, err := lastAppliedConfiguration(obj); err != nil {
		return errors.Wrapf(err, "could not generate apply annotation for resource %q", gvk)
	}

	if err := c.client.Create(ctx, obj); err != nil {
		_, resourceDescription := nameDescription(obj)
		return errors.Wrapf(err, "could not create %q", resourceDescription)
	}

	return nil
}

// ApplyCluster applies a patch to the cluster-scoped resource to move from currentState to intendedState.
func (c *ClientApplier) ApplyCluster(intendedState, currentState *unstructured.Unstructured) error {
	return c.apply("", false, intendedState, currentState)
}

// ApplyNamespace applies a patch to the namespace-scoped resource to move from currentState to intendedState.
func (c *ClientApplier) ApplyNamespace(namespace string, intendedState, currentState *unstructured.Unstructured) error {
	return c.apply(namespace, true, intendedState, currentState)
}

// apply updates a resource using the same approach as running `kubectl apply`.
// The implementation here has been mostly extracted from the apply command: k8s.io/kubernetes/pkg/kubectl/cmd/apply.go
func (c *ClientApplier) apply(namespace string, namespaceable bool, intendedState, currentState *unstructured.Unstructured) error {
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
	modified, mErr := lastAppliedConfiguration(intendedState)
	if mErr != nil {
		return errors.Errorf("could not serialize intended configuration from %v", intendedState)
	}

	gvk := intendedState.GetObjectKind().GroupVersionKind()
	resource, rErr := c.resource(gvk)
	if rErr != nil {
		return rErr
	}

	gvr := gvk.GroupVersion().WithResource(resource)
	var resourceClient dynamic.ResourceInterface
	if namespaceable {
		resourceClient = c.dynamicClient.Resource(gvr).Namespace(namespace)
	} else {
		resourceClient = c.dynamicClient.Resource(gvr)
	}

	var patch []byte
	var patchType types.PatchType

	name, resourceDescription := nameDescription(intendedState)
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
				return errors.Errorf("at least one of apiVersion, kind and name was changed for %s", resourceDescription)
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

	action.APICalls.WithLabelValues(name, "patch").Inc()
	timer := prometheus.NewTimer(action.APICallDuration.WithLabelValues(name, "patch"))
	defer timer.ObserveDuration()
	if _, err := resourceClient.Patch(name, patchType, patch); err != nil {
		return errors.Wrapf(err, "could not patch %s", resourceDescription)
	}
	glog.V(1).Infof("Patched %s", resourceDescription)
	glog.V(3).Infof("Patched with %s", patch)

	return nil
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

// lastAppliedConfiguration generates the last applied annotation from the object.
// It removes the entire labels field, if present, from the generated annotation.
// It populates the object with this annotation as well.
func lastAppliedConfiguration(original *unstructured.Unstructured) ([]byte, error) {
	// Create a copy of the object, since we will be deleting the management annotation.
	c := original.DeepCopy()
	l := c.GetAnnotations()
	delete(l, v1alpha1.ResourceManagementKey)
	c.SetAnnotations(l)

	annotation, err := kubectl.GetModifiedConfiguration(c, false, unstructured.UnstructuredJSONScheme)
	if err != nil {
		return nil, errors.Errorf("could not serialize resource into json: %v", c)
	}

	// Set the annotation on the passed in object.
	annots := original.GetAnnotations()
	annots[v1.LastAppliedConfigAnnotation] = string(annotation)
	original.SetAnnotations(annots)
	return annotation, nil
}
