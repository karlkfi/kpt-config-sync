package fake

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	"github.com/pkg/errors"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Client is a fake implementation of client.Client.
type Client struct {
	Scheme  *runtime.Scheme
	Objects map[core.ID]core.Object
}

var _ client.Client = &Client{}

// NewClient instantiates a new fake.Client pre-populated with the specified
// objects.
//
// Calls t.Fatal if unable to properly instantiate Client.
func NewClient(t *testing.T, objs ...runtime.Object) *Client {
	t.Helper()

	result := Client{
		Scheme:  runtime.NewScheme(),
		Objects: make(map[core.ID]core.Object),
	}

	err := v1.AddToScheme(result.Scheme)
	if err != nil {
		t.Fatal(errors.Wrap(err, "unable to create fake Client"))
	}

	for _, o := range objs {
		err = result.Create(context.Background(), o)
		if err != nil {
			t.Fatal(err)
		}
	}

	return &result
}

func toGR(gk schema.GroupKind) schema.GroupResource {
	return schema.GroupResource{
		Group:    gk.Group,
		Resource: gk.Kind,
	}
}

// Get implements client.Client.
func (c *Client) Get(_ context.Context, key client.ObjectKey, obj runtime.Object) error {
	co, err := core.ObjectOf(obj)
	if err != nil {
		return err
	}
	co.SetName(key.Name)
	co.SetNamespace(key.Namespace)

	if co.GroupVersionKind().Empty() {
		// Since many times we call with just an empty struct with no type metadata.
		gvks, _, err := c.Scheme.ObjectKinds(co)
		if err != nil {
			return err
		}
		switch len(gvks) {
		case 0:
			return errors.Errorf("unregistered Type; register it in fake.Client.Schema: %T", co)
		case 1:
			co.GetObjectKind().SetGroupVersionKind(gvks[0])
		default:
			return errors.Errorf("fake.Client does not support multiple Versions for the same GroupKind: %v", co)
		}
	}

	id := core.IDOf(co)
	o, ok := c.Objects[id]
	if !ok {
		return newNotFound(id)
	}

	// The actual Kubernetes implementation is much more complex.
	// This approximates the behavior, but will fail (for example) if obj lies
	// about its GroupVersionKind.
	jsn, err := json.Marshal(o)
	if err != nil {
		return errors.Wrapf(err, "unable to Marshal %v", obj)
	}
	err = json.Unmarshal(jsn, obj)
	if err != nil {
		return errors.Wrapf(err, "unable to Unmarshal: %s", string(jsn))
	}
	return nil
}

// List implements client.Client.
//
// Does not paginate results.
func (c *Client) List(_ context.Context, list runtime.Object, opts ...client.ListOption) error {
	if len(opts) > 0 {
		jsn, _ := json.MarshalIndent(opts, "", "  ")
		return errors.Errorf("fake.Client.Create does not yet support opts, but got: %v", string(jsn))
	}

	_, isList := list.(meta.List)
	if !isList {
		return errors.Errorf("called fake.Client.List on non-List type %T", list)
	}

	if ul, isUnstructured := list.(*unstructured.UnstructuredList); isUnstructured {
		return c.listUnstructured(ul)
	}

	switch l := list.(type) {
	case *v1beta1.CustomResourceDefinitionList:
		return c.listCRDs(l)
	case *v1.SyncList:
		return c.listSyncs(l)
	}

	return errors.Errorf("fake.Client does not support List(%T)", list)
}

// Create implements client.Client.
func (c *Client) Create(_ context.Context, obj runtime.Object, opts ...client.CreateOption) error {
	co, err := core.ObjectOf(obj.DeepCopyObject())
	if err != nil {
		return err
	}

	if len(opts) > 0 {
		jsn, _ := json.MarshalIndent(opts, "", "  ")
		return errors.Errorf("fake.Client.Create does not yet support opts, but got: %v", string(jsn))
	}

	id := core.IDOf(co)
	_, found := c.Objects[core.IDOf(co)]
	if found {
		return newAlreadyExists(id)
	}

	c.Objects[id] = co
	return nil
}

func validateDeleteOptions(opts []client.DeleteOption) error {
	var unsupported []client.DeleteOption
	for _, opt := range opts {
		switch opt {
		case client.PropagationPolicy(metav1.DeletePropagationBackground):
		default:
			unsupported = append(unsupported, opt)
		}
	}
	if len(unsupported) > 0 {
		jsn, _ := json.MarshalIndent(opts, "", "  ")
		return errors.Errorf("fake.Client.Delete does not yet support opts, but got: %v", string(jsn))
	}

	return nil
}

// Delete implements client.Client.
func (c *Client) Delete(_ context.Context, obj runtime.Object, opts ...client.DeleteOption) error {
	err := validateDeleteOptions(opts)
	if err != nil {
		return err
	}

	co, err := core.ObjectOf(obj.DeepCopyObject())
	if err != nil {
		return err
	}
	id := core.IDOf(co)

	_, found := c.Objects[id]
	if !found {
		return newNotFound(id)
	}
	delete(c.Objects, id)

	return nil
}

// Update implements client.Client.
func (c *Client) Update(_ context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
	if len(opts) > 0 {
		jsn, _ := json.MarshalIndent(opts, "", "  ")
		return errors.Errorf("fake.Client.Update does not yet support opts, but got: %v", string(jsn))
	}

	co, err := core.ObjectOf(obj.DeepCopyObject())
	if err != nil {
		return err
	}
	id := core.IDOf(co)

	_, found := c.Objects[id]
	if !found {
		return newNotFound(id)
	}

	c.Objects[id] = co
	return nil
}

// Patch implements client.Client.
func (c *Client) Patch(_ context.Context, obj runtime.Object, patch client.Patch, opts ...client.PatchOption) error {
	return errors.New("fake.Client does not support Patch()")
}

// DeleteAllOf implements client.Client.
func (c *Client) DeleteAllOf(_ context.Context, obj runtime.Object, opts ...client.DeleteAllOfOption) error {
	return errors.New("fake.Client does not support DeleteAllOf()")
}

// Status implements client.Client.
func (c *Client) Status() client.StatusWriter {
	return c
}

// Check reports an error to `t` if the passed objects in want do not match the
// expected set of objects in the fake.Client, and only the passed updates to
// Status fields were recorded.
func (c *Client) Check(t *testing.T, want ...runtime.Object) {
	t.Helper()

	wantMap := make(map[core.ID]core.Object)

	for _, obj := range want {
		cobj, ok := obj.(core.Object)
		if !ok {
			t.Errorf("obj is not a Kubernetes object %v", obj)
		}
		wantMap[core.IDOf(cobj)] = cobj
	}

	if diff := cmp.Diff(wantMap, c.Objects, cmpopts.EquateEmpty()); diff != "" {
		t.Error("diff to fake.Client.Objects:\n", diff)
	}
}

func newNotFound(id core.ID) error {
	return apierrors.NewNotFound(toGR(id.GroupKind), id.ObjectKey.String())
}

func newAlreadyExists(id core.ID) error {
	return apierrors.NewAlreadyExists(toGR(id.GroupKind), id.ObjectKey.String())
}

func (c *Client) list(gk schema.GroupKind) []core.Object {
	var result []core.Object
	for _, o := range c.Objects {
		if o.GroupVersionKind().GroupKind() != gk {
			continue
		}
		result = append(result, o)
	}
	return result
}

func (c *Client) listCRDs(list *v1beta1.CustomResourceDefinitionList) error {
	objs := c.list(kinds.CustomResourceDefinition())
	for _, o := range objs {
		crd, ok := o.(*v1beta1.CustomResourceDefinition)
		if !ok {
			return errors.Errorf("non-CRD stored as CRD: %v", o)
		}
		list.Items = append(list.Items, *crd)
	}

	return nil
}

func (c *Client) listSyncs(list *v1.SyncList) error {
	objs := c.list(kinds.Sync().GroupKind())
	for _, o := range objs {
		sync, ok := o.(*v1.Sync)
		if !ok {
			return errors.Errorf("non-Sync stored as CRD: %v", o)
		}
		list.Items = append(list.Items, *sync)
	}

	return nil
}

func (c *Client) listUnstructured(list *unstructured.UnstructuredList) error {
	gvk := list.GetObjectKind().GroupVersionKind()
	if gvk.Empty() {
		return errors.Errorf("fake.Client.List(UnstructuredList) requires GVK")
	}
	if !strings.HasSuffix(gvk.Kind, "List") {
		return errors.Errorf("fake.Client.List(UnstructuredList) called with non-List GVK %q", gvk.String())
	}
	gvk.Kind = strings.TrimSuffix(gvk.Kind, "List")

	for _, o := range c.list(gvk.GroupKind()) {
		obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(o)
		if err != nil {
			return err
		}
		list.Items = append(list.Items, unstructured.Unstructured{Object: obj})
	}
	return nil
}
