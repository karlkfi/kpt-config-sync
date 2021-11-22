package applier

import (
	"context"

	"github.com/GoogleContainerTools/kpt/pkg/live"
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/metadata"
	"github.com/google/nomos/pkg/status"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/apply"
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/cli-utils/pkg/util/factory"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type kptApplier interface {
	Run(context.Context, inventory.InventoryInfo, object.UnstructuredSet, apply.Options) <-chan event.Event
}

// clientSet includes the clients required for using the apply library from cli-utils
type clientSet struct {
	kptApplier    kptApplier
	invClient     inventory.InventoryClient
	client        client.Client
	resouceClient *resourceClient
}

// newClientSet creates a clientSet object.
func newClientSet(c client.Client) (*clientSet, error) {
	kubeConfigFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	matchVersionKubeConfigFlags := util.NewMatchVersionFlags(&factory.CachingRESTClientGetter{
		Delegate: kubeConfigFlags,
	})
	f := util.NewFactory(matchVersionKubeConfigFlags)

	invClient, err := inventory.NewInventoryClient(f, live.WrapInventoryObj, live.InvToUnstructuredFunc)
	if err != nil {
		return nil, err
	}
	applier, err := apply.NewApplier(f, invClient)
	if err != nil {
		return nil, err
	}

	dy, err := f.DynamicClient()
	if err != nil {
		return nil, err
	}
	mapper, err := f.ToRESTMapper()
	if err != nil {
		return nil, err
	}
	resourceClient := newResourceClient(dy, mapper)

	return &clientSet{
		kptApplier:    applier,
		invClient:     invClient,
		client:        c,
		resouceClient: resourceClient,
	}, nil
}

func (cs *clientSet) apply(ctx context.Context, inv inventory.InventoryInfo, resources []*unstructured.Unstructured, option apply.Options) <-chan event.Event {
	return cs.kptApplier.Run(ctx, inv, object.UnstructuredSet(resources), option)
}

// handleDisabledObjects remove the specified objects from the inventory, and then disable them
// one by one by removing the nomos metadata.
// It returns the number of objects which are disabled successfully, and the errors encountered.
func (cs *clientSet) handleDisabledObjects(ctx context.Context, rg *live.InventoryResourceGroup, objs []client.Object) (uint64, status.MultiError) {
	// disabledCount tracks the number of objects which are disabled successfully
	var disabledCount uint64
	err := cs.removeFromInventory(rg, objs)
	if err != nil {
		return disabledCount, Error(err)
	}
	var errs status.MultiError
	for _, obj := range objs {
		err := cs.disableObject(ctx, obj)
		if err != nil {
			klog.Warningf("failed to disable object %v", core.IDOf(obj))
			errs = status.Append(errs, Error(err))
		} else {
			klog.V(4).Infof("disabled object %v", core.IDOf(obj))
			disabledCount++
		}
	}
	return disabledCount, errs
}

func (cs *clientSet) removeFromInventory(rg *live.InventoryResourceGroup, objs []client.Object) error {
	oldObjs, err := rg.Load()
	if err != nil {
		return err
	}
	newObjs := removeFrom(oldObjs, objs)
	err = rg.Store(newObjs)
	if err != nil {
		return err
	}
	return cs.invClient.Replace(rg, newObjs, common.DryRunNone)
}

// disableObject disables the management for a single object by removing
// the ConfigSync labels and annotations.
func (cs *clientSet) disableObject(ctx context.Context, obj client.Object) error {
	meta := objMetaFrom(obj)
	u, err := cs.resouceClient.get(ctx, meta)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if metadata.HasConfigSyncMetadata(u) {
		updated := metadata.RemoveConfigSyncMetadata(u)
		if !updated {
			return nil
		}
		// APIService is handled specially by client-side apply due to
		// https://github.com/kubernetes/kubernetes/issues/89264
		if u.GroupVersionKind().GroupKind() == kinds.APIService().GroupKind() {
			err = updateLabelsAndAnnotations(u, u.GetLabels(), u.GetAnnotations())
			if err != nil {
				return err
			}
			return cs.resouceClient.update(ctx, meta, u, nil)
		}
		u.SetManagedFields(nil)
		return cs.client.Patch(ctx, u, client.Apply, client.FieldOwner(configsync.FieldManager), client.ForceOwnership)
	}
	return nil
}
