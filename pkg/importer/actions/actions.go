package actions

import (
	"time"

	typedv1 "github.com/google/nomos/clientgen/apis/typed/configmanagement/v1"
	listersv1 "github.com/google/nomos/clientgen/listers/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/importer"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/clusterconfig"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"github.com/google/nomos/pkg/util/sync"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Factories contains factories for creating actions on Nomos custom resources.
type Factories struct {
	NamespaceConfigAction namespaceConfigActionFactory
	ClusterConfigAction   clusterConfigActionFactory
	SyncAction            syncActionFactory
}

// NewFactories creates a new Factories.
func NewFactories(
	v1client typedv1.ConfigmanagementV1Interface, v1alpha1client typedv1.ConfigmanagementV1Interface,
	pnLister listersv1.NamespaceConfigLister, cpLister listersv1.ClusterConfigLister,
	syncLister listersv1.SyncLister) Factories {
	return Factories{newNamespaceConfigActionFactory(v1client, pnLister),
		newClusterConfigActionFactory(v1client, cpLister),
		newSyncActionFactory(v1alpha1client, syncLister)}
}

type namespaceConfigActionFactory struct {
	*action.ReflectiveActionSpec
}

func newNamespaceConfigActionFactory(client typedv1.ConfigmanagementV1Interface, lister listersv1.NamespaceConfigLister) namespaceConfigActionFactory {
	return namespaceConfigActionFactory{namespaceconfig.NewActionSpec(client, lister)}
}

// NewCreate returns an action for creating NamespaceConfigs.
func (f namespaceConfigActionFactory) NewCreate(namespaceConfig *v1.NamespaceConfig) action.Interface {
	onExecute := func(duration float64, err error) {
		importer.Metrics.Operations.WithLabelValues("create", "namespace", statusLabel(err)).Inc()
		importer.Metrics.APICallDuration.WithLabelValues("create", "namespace", statusLabel(err)).Observe(duration)
	}
	return action.NewReflectiveCreateAction("", namespaceConfig.Name, namespaceConfig, f.ReflectiveActionSpec, onExecute)
}

// NewUpdate returns an action for updating NamespaceConfigs. This action ignores the ResourceVersion of
// the new NamespaceConfig as well as most of the Status. If Status.SyncState has been set then that will
// be copied over.
func (f namespaceConfigActionFactory) NewUpdate(namespaceConfig *v1.NamespaceConfig) action.Interface {
	updateConfig := func(old runtime.Object) (runtime.Object, error) {
		newPN := namespaceConfig.DeepCopy()
		oldPN := old.(*v1.NamespaceConfig)
		if !oldPN.Spec.DeleteSyncedTime.IsZero() {
			e := status.ResourceWrap(errors.Errorf("namespace %v terminating, cannot update", oldPN.Name), "", ast.ParseFileObject(old))
			return nil, e
		}
		newPN.ResourceVersion = oldPN.ResourceVersion
		newSyncState := newPN.Status.SyncState
		oldPN.Status.DeepCopyInto(&newPN.Status)
		if !newSyncState.IsUnknown() {
			newPN.Status.SyncState = newSyncState
		}
		return newPN, nil
	}
	onExecute := func(duration float64, err error) {
		importer.Metrics.Operations.WithLabelValues("update", "namespace", statusLabel(err)).Inc()
		importer.Metrics.APICallDuration.WithLabelValues("update", "namespace", statusLabel(err)).Observe(duration)
	}
	return action.NewReflectiveUpdateAction("", namespaceConfig.Name, updateConfig, f.ReflectiveActionSpec, onExecute)
}

// NewDelete returns an action for deleting NamespaceConfigs.
func (f namespaceConfigActionFactory) NewDelete(nodeName string, configs namespaceconfig.AllConfigs) action.Interface {
	onExecute := func(duration float64, err error) {
		importer.Metrics.Operations.WithLabelValues("delete", "namespace", statusLabel(err)).Inc()
		importer.Metrics.APICallDuration.WithLabelValues("delete", "namespace", statusLabel(err)).Observe(duration)
	}
	updateConfig := func(old runtime.Object) (runtime.Object, error) {
		newNsConf := old.(*v1.NamespaceConfig).DeepCopy()
		newNsConf.Spec.DeleteSyncedTime = metav1.Now()
		// During a delete, the nsconfig is not present in the repo, so the outputvisitor can't update
		// sync info during traversal; so, update here instead
		newNsConf.Spec.Token = configs.ImportToken
		newNsConf.Spec.ImportTime = metav1.NewTime(configs.LoadTime)
		return newNsConf, nil
	}
	return action.NewReflectiveUpdateAction("", nodeName, updateConfig, f.ReflectiveActionSpec, onExecute)
}

type clusterConfigActionFactory struct {
	*action.ReflectiveActionSpec
}

func newClusterConfigActionFactory(
	client typedv1.ConfigmanagementV1Interface,
	lister listersv1.ClusterConfigLister) clusterConfigActionFactory {
	return clusterConfigActionFactory{clusterconfig.NewActionSpec(client, lister)}
}

// NewCreate returns an action for creating ClusterConfigs.
func (f clusterConfigActionFactory) NewCreate(clusterConfig *v1.ClusterConfig) action.Interface {
	onExecute := func(duration float64, err error) {
		importer.Metrics.Operations.WithLabelValues("create", "cluster", statusLabel(err)).Inc()
		importer.Metrics.APICallDuration.WithLabelValues("create", "cluster", statusLabel(err)).Observe(duration)
	}
	return action.NewReflectiveCreateAction("", clusterConfig.Name, clusterConfig, f.ReflectiveActionSpec, onExecute)
}

// NewUpdate returns an action for updating ClusterConfigs. This action ignores the ResourceVersion
// of the new ClusterConfig as well as most of the Status. If Status.SyncState has been set then
// that will be copied over.
func (f clusterConfigActionFactory) NewUpdate(clusterConfig *v1.ClusterConfig) action.Interface {
	updateConfig := func(old runtime.Object) (runtime.Object, error) {
		newCP := clusterConfig.DeepCopy()
		oldCP := old.(*v1.ClusterConfig)
		newCP.ResourceVersion = oldCP.ResourceVersion
		newSyncState := newCP.Status.SyncState
		oldCP.Status.DeepCopyInto(&newCP.Status)
		if !newSyncState.IsUnknown() {
			newCP.Status.SyncState = newSyncState
		}
		return newCP, nil
	}
	onExecute := func(duration float64, err error) {
		importer.Metrics.Operations.WithLabelValues("update", "cluster", statusLabel(err)).Inc()
		importer.Metrics.APICallDuration.WithLabelValues("update", "cluster", statusLabel(err)).Observe(duration)
	}
	return action.NewReflectiveUpdateAction("", clusterConfig.Name, updateConfig, f.ReflectiveActionSpec, onExecute)
}

// NewDelete returns an action for deleting ClusterConfigs.
func (f clusterConfigActionFactory) NewDelete(clusterConfigName string) action.Interface {
	onExecute := func(duration float64, err error) {
		importer.Metrics.Operations.WithLabelValues("delete", "cluster", statusLabel(err)).Inc()
		importer.Metrics.APICallDuration.WithLabelValues("delete", "cluster", statusLabel(err)).Observe(duration)
	}
	return action.NewReflectiveDeleteAction("", clusterConfigName, f.ReflectiveActionSpec, onExecute)
}

type syncActionFactory struct {
	*action.ReflectiveActionSpec
}

func newSyncActionFactory(
	client typedv1.ConfigmanagementV1Interface,
	lister listersv1.SyncLister) syncActionFactory {
	return syncActionFactory{sync.NewActionSpec(client, lister)}
}

func (f syncActionFactory) NewCreate(sync v1.Sync) action.Interface {
	onExecute := func(duration float64, err error) {
		importer.Metrics.Operations.WithLabelValues("create", "sync", statusLabel(err)).Inc()
		importer.Metrics.APICallDuration.WithLabelValues("create", "sync", statusLabel(err)).Observe(duration)
	}
	return action.NewReflectiveCreateAction("", sync.Name, &sync, f.ReflectiveActionSpec, onExecute)
}

func (f syncActionFactory) NewUpdate(sync v1.Sync) action.Interface {
	updateSync := func(old runtime.Object) (runtime.Object, error) {
		newSync := sync.DeepCopy()
		oldSync := old.(*v1.Sync)
		newSync.ResourceVersion = oldSync.ResourceVersion
		return newSync, nil
	}
	onExecute := func(duration float64, err error) {
		importer.Metrics.Operations.WithLabelValues("update", "sync", statusLabel(err)).Inc()
		importer.Metrics.APICallDuration.WithLabelValues("update", "sync", statusLabel(err)).Observe(duration)
	}
	return action.NewReflectiveUpdateAction("", sync.Name, updateSync, f.ReflectiveActionSpec, onExecute)
}

func (f syncActionFactory) NewDelete(syncName string, timeout time.Duration) action.Interface {
	onExecute := func(duration float64, err error) {
		importer.Metrics.Operations.WithLabelValues("delete", "sync", statusLabel(err)).Inc()
		importer.Metrics.APICallDuration.WithLabelValues("delete", "sync", statusLabel(err)).Observe(duration)
	}
	return action.NewBlockingReflectiveDeleteAction("", syncName, timeout, f.ReflectiveActionSpec, onExecute)
}

func statusLabel(err error) string {
	if err == nil {
		return "success"
	}
	return "error"

}
