package fake

import (
	"context"
	"encoding/json"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SyncCacheReader is a fake client.Reader for testing.
type SyncCacheReader v1.SyncList

var _ client.Reader = SyncCacheReader{}

// Get implements Reader.
func (r SyncCacheReader) Get(_ context.Context, key client.ObjectKey, obj runtime.Object) error {
	return errors.New("fake.SyncCacheReader.Get is not implemented")
}

// List implements Reader.
func (r SyncCacheReader) List(_ context.Context, list runtime.Object, opts ...client.ListOption) error {
	result, ok := list.(*v1.SyncList)
	if !ok {
		return errors.Errorf("list passed to SyncCachedReader must be a *v1.SyncList, but was %T", list)
	}
	if len(opts) > 0 {
		jsn, _ := json.MarshalIndent(opts, "", "  ")
		return errors.Errorf("fake.SyncCacheReader.List does not yet support opts, but got: %v", string(jsn))
	}

	result.TypeMeta = r.TypeMeta
	r.ListMeta.DeepCopyInto(&result.ListMeta)
	result.Items = make([]v1.Sync, len(r.Items))
	for i, o := range r.Items {
		o.DeepCopyInto(&result.Items[i])
	}

	return nil
}
