package differ

import (
	"context"

	"github.com/google/nomos/pkg/syncer/decode"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/status"
	syncerclient "github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/util/clusterconfig"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"github.com/google/nomos/pkg/util/sync"
)

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
				_, err := d.client.Update(ctx, &intent, func(obj runtime.Object) (runtime.Object, error) {
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
			d.errs = status.Append(d.errs, d.client.Delete(ctx, &mayDelete))
			deletes++
		}
	}

	glog.Infof("NamespaceConfig operations: create %d, update %d, delete %d", creates, updates, deletes)
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

	equal, err := clusterconfig.ClusterConfigsEqual(decoder, desired, current)
	if err != nil {
		d.errs = status.Append(d.errs, err)
		return
	}
	if !equal {
		_, err := d.client.Update(ctx, desired, func(obj runtime.Object) (runtime.Object, error) {
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
			if !sync.SyncsEqual(&newSync, &oldSync) {
				_, err := d.client.Update(ctx, &newSync, func(obj runtime.Object) (runtime.Object, error) {
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
