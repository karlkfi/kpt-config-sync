package differ

import (
	"context"

	"github.com/golang/glog"
	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/status"
	syncerclient "github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/decode"
	"github.com/google/nomos/pkg/util/compare"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Alias metav1.Now to enable test mocking.
var now = metav1.Now

// Update updates the nomos CRs on the cluster based on the difference between the desired and current state.
func Update(ctx context.Context, client *syncerclient.Client, decoder decode.Decoder, current, desired namespaceconfig.AllConfigs) status.MultiError {
	var errs status.MultiError
	impl := &differ{client: client, errs: errs}
	impl.updateNamespaceConfigs(ctx, decoder, current, desired)
	impl.updateClusterConfigs(ctx, decoder, current, desired)
	impl.updateSyncs(ctx, current, desired)
	return impl.errs
}

// differ does the actual update operations on the cluster and keeps track of the errors encountered while doing so.
type differ struct {
	client *syncerclient.Client
	errs   status.MultiError
}

// updateNamespaceConfigs compares the given sets of current and desired NamespaaceConfigs and performs the necessary
// actions to update the current set to match the desired set.
func (d *differ) updateNamespaceConfigs(ctx context.Context, decoder decode.Decoder, current, desired namespaceconfig.AllConfigs) {
	var deletes, creates, updates int
	for name := range desired.NamespaceConfigs {
		intent := desired.NamespaceConfigs[name]
		if actual, found := current.NamespaceConfigs[name]; found {
			equal, err := namespaceconfig.NamespaceConfigsEqual(decoder, &intent, &actual)
			if err != nil {
				d.errs = status.Append(d.errs, err)
				continue
			}
			if !equal {
				err := d.updateNamespaceConfig(ctx, &intent)
				d.errs = status.Append(d.errs, err)
				updates++
			}
		} else {
			d.errs = status.Append(d.errs, d.client.Create(ctx, &intent))
			creates++
		}
	}
	for name, mayDelete := range current.NamespaceConfigs {
		if _, found := desired.NamespaceConfigs[name]; !found {
			d.errs = status.Append(d.errs, d.deleteNamespaceConfig(ctx, &mayDelete))
			deletes++
		}
	}

	glog.Infof("NamespaceConfig operations: create %d, update %d, delete %d", creates, updates, deletes)
}

// deleteNamespaceConfig marks the given NamespaceConfig for deletion by the syncer. This tombstone is more explicit
// than having the importer just delete the NamespaceConfig directly.
func (d *differ) deleteNamespaceConfig(ctx context.Context, nc *v1.NamespaceConfig) status.Error {
	_, err := d.client.Update(ctx, nc, func(obj core.Object) (core.Object, error) {
		newObj := obj.(*v1.NamespaceConfig).DeepCopy()
		newObj.Spec.DeleteSyncedTime = now()
		return newObj, nil
	})
	return err
}

// updateNamespaceConfig writes the given NamespaceConfig to storage as it is specified.
func (d *differ) updateNamespaceConfig(ctx context.Context, intent *v1.NamespaceConfig) status.Error {
	_, err := d.client.Update(ctx, intent, func(obj core.Object) (core.Object, error) {
		oldObj := obj.(*v1.NamespaceConfig)
		newObj := intent.DeepCopy()
		if !oldObj.Spec.DeleteSyncedTime.IsZero() {
			e := status.ResourceWrap(errors.Errorf("namespace %v terminating, cannot update", oldObj.Name), "",
				ast.ParseFileObject(oldObj))
			return nil, e
		}
		newObj.ResourceVersion = oldObj.ResourceVersion
		newSyncState := newObj.Status.SyncState
		oldObj.Status.DeepCopyInto(&newObj.Status)
		if !newSyncState.IsUnknown() {
			newObj.Status.SyncState = newSyncState
		}
		return newObj, nil
	})
	return err
}

func (d *differ) updateClusterConfigs(ctx context.Context, decoder decode.Decoder, current, desired namespaceconfig.AllConfigs) {
	d.updateClusterConfig(ctx, decoder, current.ClusterConfig, desired.ClusterConfig)
	d.updateClusterConfig(ctx, decoder, current.CRDClusterConfig, desired.CRDClusterConfig)
}

func (d *differ) updateClusterConfig(ctx context.Context, decoder decode.Decoder, current, desired *v1.ClusterConfig) {
	if current == nil && desired == nil {
		return
	}
	if current == nil {
		d.errs = status.Append(d.errs, d.client.Create(ctx, desired))
		return
	}
	if desired == nil {
		d.errs = status.Append(d.errs, d.client.Delete(ctx, current))
		return
	}

	equal, err := compare.GenericResourcesEqual(decoder, desired.Spec.Resources, current.Spec.Resources)
	if err != nil {
		d.errs = status.Append(d.errs, err)
		return
	}
	if !equal {
		_, err := d.client.Update(ctx, desired, func(obj core.Object) (core.Object, error) {
			oldObj := obj.(*v1.ClusterConfig)
			newObj := desired.DeepCopy()
			newObj.ResourceVersion = oldObj.ResourceVersion
			newSyncState := newObj.Status.SyncState
			oldObj.Status.DeepCopyInto(&newObj.Status)
			if !newSyncState.IsUnknown() {
				newObj.Status.SyncState = newSyncState
			}
			return newObj, nil
		})
		d.errs = status.Append(d.errs, err)
	}
}

func (d *differ) updateSyncs(ctx context.Context, current, desired namespaceconfig.AllConfigs) {
	var creates, updates, deletes int
	for name, newSync := range desired.Syncs {
		if oldSync, exists := current.Syncs[name]; exists {
			if !syncsEqual(&newSync, &oldSync) {
				_, err := d.client.Update(ctx, &newSync, func(obj core.Object) (core.Object, error) {
					oldObj := obj.(*v1.Sync)
					newObj := newSync.DeepCopy()
					newObj.ResourceVersion = oldObj.ResourceVersion
					return newObj, nil
				})
				d.errs = status.Append(d.errs, err)
				updates++
			}
		} else {
			d.errs = status.Append(d.errs, d.client.Create(ctx, &newSync))
			creates++
		}
	}

	for name, mayDelete := range current.Syncs {
		if _, found := desired.Syncs[name]; !found {
			d.errs = status.Append(d.errs, d.client.Delete(ctx, &mayDelete))
			deletes++
		}
	}

	glog.Infof("Sync operations: %d updates, %d creates, %d deletes", updates, creates, deletes)
}

// syncsEqual returns true if the syncs are equivalent.
func syncsEqual(l *v1.Sync, r *v1.Sync) bool {
	return cmp.Equal(l.Spec, r.Spec) && compare.ObjectMetaEqual(l, r)
}
