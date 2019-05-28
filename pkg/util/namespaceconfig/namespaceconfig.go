package namespaceconfig

import (
	"context"

	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ListConfigs returns all configs from API server.
func ListConfigs(ctx context.Context, cache cache.Cache) (*AllConfigs, error) {
	configs := AllConfigs{
		NamespaceConfigs: make(map[string]v1.NamespaceConfig),
	}

	// NamespaceConfigs
	namespaceConfigs := &v1.NamespaceConfigList{}
	if err := cache.List(ctx, &client.ListOptions{}, namespaceConfigs); err != nil {
		return nil, errors.Wrap(err, "failed to list NamespaceConfigs")
	}
	for _, n := range namespaceConfigs.Items {
		configs.NamespaceConfigs[n.Name] = *n.DeepCopy()
	}

	// ClusterConfigs
	if cErr := DecorateWithClusterConfigs(ctx, cache, &configs); cErr != nil {
		return nil, errors.Wrap(cErr, "failed to list ClusterConfigs")
	}

	// Syncs
	var err error
	configs.Syncs, err = ListSyncs(ctx, cache)
	return &configs, err
}

// DecorateWithClusterConfigs updates AllPolices with all the ClusterConfigs from APIServer.
func DecorateWithClusterConfigs(ctx context.Context, cache client.Reader, policies *AllConfigs) error {
	clusterConfigs := &v1.ClusterConfigList{}
	if err := cache.List(ctx, &client.ListOptions{}, clusterConfigs); err != nil {
		return errors.Wrap(err, "failed to list ClusterConfigs")
	}

	for _, c := range clusterConfigs.Items {
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
// the cache.
func ListSyncs(ctx context.Context, cache cache.Cache) (map[string]v1.Sync, error) {
	syncs := &v1.SyncList{}
	if err := cache.List(ctx, &client.ListOptions{}, syncs); err != nil {
		return nil, errors.Wrap(err, "failed to list Syncs")
	}

	var ret map[string]v1.Sync
	if len(syncs.Items) > 0 {
		ret = make(map[string]v1.Sync, len(syncs.Items))
	}

	for _, s := range syncs.Items {
		ret[s.Name] = *s.DeepCopy()
	}
	return ret, nil
}
