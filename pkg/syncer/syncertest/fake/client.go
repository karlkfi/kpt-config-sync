package fake

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/syncer/reconcile"
	"github.com/google/nomos/pkg/util/clusterconfig"
	"github.com/pkg/errors"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
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
func NewClient(t *testing.T, scheme *runtime.Scheme, objs ...runtime.Object) *Client {
	t.Helper()

	result := Client{
		Scheme:  scheme,
		Objects: make(map[core.ID]core.Object),
	}

	err := v1.AddToScheme(result.Scheme)
	if err != nil {
		t.Fatal(errors.Wrap(err, "unable to create fake Client"))
	}

	err = v1alpha1.AddToScheme(result.Scheme)
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

func validateListOptions(opts client.ListOptions) error {
	if opts.Continue != "" {
		return errors.Errorf("fake.Client.List does not yet support the Continue option, but got: %+v", opts)
	}
	if opts.FieldSelector != nil {
		return errors.Errorf("fake.Client.List does not yet support the FieldSelector option, but got: %+v", opts)
	}
	if opts.Limit != 0 {
		return errors.Errorf("fake.Client.List does not yet support the Limit option, but got: %+v", opts)
	}

	return nil
}

// List implements client.Client.
//
// Does not paginate results.
func (c *Client) List(_ context.Context, list runtime.Object, opts ...client.ListOption) error {
	options := client.ListOptions{}
	options.ApplyOptions(opts)
	err := validateListOptions(options)
	if err != nil {
		return err
	}

	_, isList := list.(meta.List)
	if !isList {
		return errors.Errorf("called fake.Client.List on non-List type %T", list)
	}

	if ul, isUnstructured := list.(*unstructured.UnstructuredList); isUnstructured {
		return c.listUnstructured(ul, options)
	}

	switch l := list.(type) {
	case *v1beta1.CustomResourceDefinitionList:
		return c.listCRDs(l, options)
	case *v1.SyncList:
		return c.listSyncs(l, options)
	}

	return errors.Errorf("fake.Client does not support List(%T)", list)
}

func (c *Client) fromUnstructured(obj runtime.Object) (runtime.Object, error) {
	// If possible, we want to deal with the non-Unstructured form of objects.
	// Unstructureds are prone to declare a bunch of empty maps we don't care
	// about, and can't easily tell cmp.Diff to ignore.

	u, isUnstructured := obj.(*unstructured.Unstructured)
	if !isUnstructured {
		// Already not unstructured.
		return obj, nil
	}

	result, err := c.Scheme.New(u.GroupVersionKind())
	if err != nil {
		// The type isn't registered.
		return obj, nil
	}

	jsn, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(jsn, result)
	return result, err
}

// Create implements client.Client.
func (c *Client) Create(_ context.Context, obj runtime.Object, opts ...client.CreateOption) error {
	if len(opts) > 0 {
		jsn, _ := json.MarshalIndent(opts, "", "  ")
		return errors.Errorf("fake.Client.Create does not yet support opts, but got: %+v", string(jsn))
	}

	obj, err := c.fromUnstructured(obj.DeepCopyObject())
	if err != nil {
		return err
	}

	co, err := core.ObjectOf(obj)
	if err != nil {
		return err
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

	obj, err := c.fromUnstructured(obj.DeepCopyObject())
	if err != nil {
		return err
	}

	co, err := core.ObjectOf(obj)
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
func (c *Client) Patch(ctx context.Context, obj runtime.Object, _ client.Patch, _ ...client.PatchOption) error {
	// Currently re-using the Update implementation for Patch since it fits the use-case where this is used for unit tests.
	// Please use this with caution for your use-case.
	return c.Update(ctx, obj)
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
func (c *Client) Check(t *testing.T, wants ...runtime.Object) {
	t.Helper()

	wantMap := make(map[core.ID]core.Object)

	for _, obj := range wants {
		obj, err := c.fromUnstructured(obj)
		if err != nil {
			// This is a test precondition, and if it fails the following error
			// messages will be garbage.
			t.Fatal(err)
		}

		cobj, ok := obj.(core.Object)
		if !ok {
			t.Errorf("obj is not a Kubernetes object %v", obj)
		}
		wantMap[core.IDOf(cobj)] = cobj
	}

	checked := make(map[core.ID]bool)
	for id, want := range wantMap {
		checked[id] = true
		actual, found := c.Objects[id]
		if !found {
			t.Errorf("fake.Client missing %s", id.String())
			continue
		}

		_, wantUnstructured := want.(*unstructured.Unstructured)
		_, actualUnstructured := actual.(*unstructured.Unstructured)
		if wantUnstructured != actualUnstructured {
			// If you see this error, you should register the type so the code can
			// compare them properly.
			t.Errorf("got want.(type)=%T and actual.(type)=%T for two objects of type %s, want equal", want, actual, want.GroupVersionKind().String())
			continue
		}

		if diff := cmp.Diff(want, actual, cmpopts.EquateEmpty()); diff != "" {
			// If you're seeing errors originating from how unstructured conversions work,
			// e.g. the diffs are a bunch of nil maps, then register the type in the
			// client's scheme.
			t.Errorf("diff to fake.Client.Objects[%s]:\n%s", id.String(), diff)
		}
	}
	for id := range c.Objects {
		if !checked[id] {
			t.Errorf("fake.Client unexpectedly contains %s", id.String())
		}
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

func (c *Client) listCRDs(list *v1beta1.CustomResourceDefinitionList, options client.ListOptions) error {
	objs := c.list(kinds.CustomResourceDefinition())
	for _, obj := range objs {
		if options.Namespace != "" && obj.GetNamespace() != options.Namespace {
			continue
		}
		if options.LabelSelector != nil {
			l := labels.Set(obj.GetLabels())
			if !options.LabelSelector.Matches(l) {
				continue
			}
		}
		switch o := obj.(type) {
		// TODO(b/154527698): Handle v1.CRDs once we're able to import the definition.
		case *v1beta1.CustomResourceDefinition:
			list.Items = append(list.Items, *o)
		case *unstructured.Unstructured:
			crd, err := clusterconfig.AsCRD(o)
			if err != nil {
				return err
			}
			list.Items = append(list.Items, *crd)
		default:
			return errors.Errorf("non-CRD stored as CRD: %+v", obj)
		}
	}

	return nil
}

func (c *Client) listSyncs(list *v1.SyncList, options client.ListOptions) error {
	objs := c.list(kinds.Sync().GroupKind())
	for _, obj := range objs {
		if options.Namespace != "" && obj.GetNamespace() != options.Namespace {
			continue
		}
		if options.LabelSelector != nil {
			l := labels.Set(obj.GetLabels())
			if !options.LabelSelector.Matches(l) {
				continue
			}
		}
		sync, ok := obj.(*v1.Sync)
		if !ok {
			return errors.Errorf("non-Sync stored as CRD: %v", obj)
		}
		list.Items = append(list.Items, *sync)
	}

	return nil
}

func (c *Client) listUnstructured(list *unstructured.UnstructuredList, options client.ListOptions) error {
	gvk := list.GetObjectKind().GroupVersionKind()
	if gvk.Empty() {
		return errors.Errorf("fake.Client.List(UnstructuredList) requires GVK")
	}
	if !strings.HasSuffix(gvk.Kind, "List") {
		return errors.Errorf("fake.Client.List(UnstructuredList) called with non-List GVK %q", gvk.String())
	}
	gvk.Kind = strings.TrimSuffix(gvk.Kind, "List")

	for _, obj := range c.list(gvk.GroupKind()) {
		if options.Namespace != "" && obj.GetNamespace() != options.Namespace {
			continue
		}
		if options.LabelSelector != nil {
			l := labels.Set(obj.GetLabels())
			if !options.LabelSelector.Matches(l) {
				continue
			}
		}
		uo, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return err
		}
		list.Items = append(list.Items, unstructured.Unstructured{Object: uo})
	}
	return nil
}

// Applier returns a fake.Applier wrapping this fake.Client. Callers using the
// resulting Applier will read from/write to the original fake.Client.
func (c *Client) Applier() reconcile.Applier {
	return &applier{Client: c}
}
