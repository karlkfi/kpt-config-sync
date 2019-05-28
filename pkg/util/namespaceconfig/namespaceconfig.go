package namespaceconfig

import (
	listersv1 "github.com/google/nomos/clientgen/listers/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/labels"
)

// ListConfigs returns all configs from API server.
func ListConfigs(namespaceConfigLister listersv1.NamespaceConfigLister,
	clusterConfigLister listersv1.ClusterConfigLister,
	syncLister listersv1.SyncLister) (*AllConfigs, error) {
	configs := AllConfigs{
		NamespaceConfigs: make(map[string]v1.NamespaceConfig),
	}

	// NamespaceConfigs
	pn, err := namespaceConfigLister.List(labels.Everything())
	if err != nil {
		return nil, errors.Wrap(err, "failed to list NamespaceConfigs")
	}
	for _, n := range pn {
		configs.NamespaceConfigs[n.Name] = *n.DeepCopy()
	}

	// ClusterConfigs
	if cErr := DecorateWithClusterConfigs(clusterConfigLister, &configs); cErr != nil {
		return nil, errors.Wrap(cErr, "failed to list ClusterConfigs")
	}

	// Syncs
	configs.Syncs, err = ListSyncs(syncLister)
	return &configs, err
}

// DecorateWithClusterConfigs updates AllPolices with all the ClusterConfigs from APIServer.
func DecorateWithClusterConfigs(lister listersv1.ClusterConfigLister, policies *AllConfigs) error {
	cp, err := lister.List(labels.Everything())
	if err != nil {
		return errors.Wrap(err, "failed to list ClusterConfigs")
	}
	for _, c := range cp {
		switch n := c.Name; n {
		case v1.ClusterConfigName:
			policies.ClusterConfig = c.DeepCopy()
		case v1.CRDClusterConfigName:
			policies.CRDClusterConfig = c.DeepCopy()
		default:
			return errors.Errorf("found an invalid ClusterConfig: %s", n)
		}
	}
	return nil
}

// ListSyncs gets a map-by-name of Syncs currently present in the cluster from
// the provided lister.
func ListSyncs(syncLister listersv1.SyncLister) (ret map[string]v1.Sync, err error) {
	syncs, err := syncLister.List(labels.Everything())
	if err != nil {
		return nil, errors.Wrap(err, "failed to list Syncs")
	}
	if len(syncs) > 0 {
		ret = make(map[string]v1.Sync, len(syncs))
	}
	for _, s := range syncs {
		ret[s.Name] = *s.DeepCopy()
	}
	return ret, err
}
