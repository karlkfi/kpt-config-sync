/*
Copyright 2018 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package actions

import (
	"time"

	typedv1 "github.com/google/nomos/clientgen/apis/typed/configmanagement/v1"
	listersv1 "github.com/google/nomos/clientgen/listers/configmanagement/v1"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/util/clusterconfig"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"github.com/google/nomos/pkg/util/sync"
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
	return action.NewReflectiveCreateAction("", namespaceConfig.Name, namespaceConfig, f.ReflectiveActionSpec)
}

// NewUpdate returns an action for updating NamespaceConfigs. This action ignores the ResourceVersion of
// the new NamespaceConfig as well as most of the Status. If Status.SyncState has been set then that will
// be copied over.
func (f namespaceConfigActionFactory) NewUpdate(namespaceConfig *v1.NamespaceConfig) action.Interface {
	updatePolicy := func(old runtime.Object) (runtime.Object, error) {
		newPN := namespaceConfig.DeepCopy()
		oldPN := old.(*v1.NamespaceConfig)
		newPN.ResourceVersion = oldPN.ResourceVersion
		newSyncState := newPN.Status.SyncState
		oldPN.Status.DeepCopyInto(&newPN.Status)
		if !newSyncState.IsUnknown() {
			newPN.Status.SyncState = newSyncState
		}
		return newPN, nil
	}
	return action.NewReflectiveUpdateAction("", namespaceConfig.Name, updatePolicy, f.ReflectiveActionSpec)
}

// NewDelete returns an action for deleting NamespaceConfigs.
func (f namespaceConfigActionFactory) NewDelete(nodeName string) action.Interface {
	return action.NewReflectiveDeleteAction("", nodeName, f.ReflectiveActionSpec)
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
	return action.NewReflectiveCreateAction("", clusterConfig.Name, clusterConfig, f.ReflectiveActionSpec)
}

// NewUpdate returns an action for updating ClusterConfigs. This action ignores the ResourceVersion
// of the new ClusterConfig as well as most of the Status. If Status.SyncState has been set then
// that will be copied over.
func (f clusterConfigActionFactory) NewUpdate(clusterConfig *v1.ClusterConfig) action.Interface {
	updatePolicy := func(old runtime.Object) (runtime.Object, error) {
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
	return action.NewReflectiveUpdateAction("", clusterConfig.Name, updatePolicy, f.ReflectiveActionSpec)
}

// NewDelete returns an action for deleting ClusterConfigs.
func (f clusterConfigActionFactory) NewDelete(
	clusterConfigName string) action.Interface {
	return action.NewReflectiveDeleteAction("", clusterConfigName, f.ReflectiveActionSpec)
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
	return action.NewReflectiveCreateAction("", sync.Name, &sync, f.ReflectiveActionSpec)
}

func (f syncActionFactory) NewUpdate(sync v1.Sync) action.Interface {
	updateSync := func(old runtime.Object) (runtime.Object, error) {
		newSync := sync.DeepCopy()
		oldSync := old.(*v1.Sync)
		newSync.ResourceVersion = oldSync.ResourceVersion
		return newSync, nil
	}
	return action.NewReflectiveUpdateAction("", sync.Name, updateSync, f.ReflectiveActionSpec)
}

func (f syncActionFactory) NewDelete(syncName string, timeout time.Duration) action.Interface {
	return action.NewBlockingReflectiveDeleteAction("", syncName, timeout, f.ReflectiveActionSpec)
}
