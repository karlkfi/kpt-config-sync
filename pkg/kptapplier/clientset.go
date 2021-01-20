package kptapplier

import (
	"context"

	kptclient "github.com/GoogleContainerTools/kpt/pkg/client"
	"github.com/GoogleContainerTools/kpt/pkg/live"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/differ"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/apply"
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/util/factory"
)

type kptApplier interface {
	Run(context.Context, inventory.InventoryInfo, []*unstructured.Unstructured, apply.Options) <-chan event.Event
}

type clientSet struct {
	kptApplier    kptApplier
	invClient     inventory.InventoryClient
	resouceClient *kptclient.Client
}

func newClientSet() (*clientSet, error) {
	kubeConfigFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	matchVersionKubeConfigFlags := util.NewMatchVersionFlags(&factory.CachingRESTClientGetter{
		Delegate: kubeConfigFlags,
	})
	f := util.NewFactory(matchVersionKubeConfigFlags)
	provider := live.NewResourceGroupProvider(f)
	applier := apply.NewApplier(provider)
	invClient, err := provider.InventoryClient()
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
	resourceClient := kptclient.NewClient(dy, mapper)
	// TODO(b/177469646): The applier currently needs to be re-initialized
	// to capture the new applied CRDs. We can optimize this by
	// avoiding unnecessary re-initialization.
	err = applier.Initialize()
	if err != nil {
		return nil, err
	}
	return &clientSet{
		kptApplier:    applier,
		invClient:     invClient,
		resouceClient: resourceClient,
	}, nil
}

func (cs *clientSet) apply(ctx context.Context, inv inventory.InventoryInfo, resources []*unstructured.Unstructured, option apply.Options) <-chan event.Event {
	return cs.kptApplier.Run(ctx, inv, resources, option)
}

func (cs *clientSet) handleDisabledObjects(ctx context.Context, inv inventory.InventoryInfo, objs []core.Object) status.MultiError {
	err := cs.removeFromInventory(inv, objs)
	if err != nil {
		return ApplierError(err)
	}
	var errs status.MultiError
	for _, obj := range objs {
		err := cs.disableObject(ctx, obj)
		if err != nil {
			errs = status.Append(errs, ApplierError(err))
		}
	}
	return errs
}

func (cs *clientSet) removeFromInventory(inv inventory.InventoryInfo, objs []core.Object) error {
	// TODO(b/178016280): change a.inventory to *live.InventoryResourceGroup.
	rg := inv.(*live.InventoryResourceGroup)
	oldObjs, err := rg.Load()
	if err != nil {
		return err
	}
	newObjs := removeFrom(oldObjs, objs)
	err = rg.Store(newObjs)
	if err != nil {
		return err
	}
	return cs.invClient.Replace(inv, newObjs)
}

// disableObject disables the management for a single object by removing
// the ConfigSync labels and annotations and adding the annotation for disabled management.
func (cs *clientSet) disableObject(ctx context.Context, obj core.Object) error {
	meta := objMetaFrom(obj)
	u, err := cs.resouceClient.Get(ctx, meta)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if differ.HasNomosMeta(u) {
		labels, annotations, updated := removeConfigSyncLabelsAndAnnotations(u)
		if !updated {
			return nil
		}
		err = kptclient.UpdateLabelsAndAnnotations(u, labels, annotations)
		if err != nil {
			return err
		}
		return cs.resouceClient.Update(ctx, meta, u, nil)
	}
	return nil
}
