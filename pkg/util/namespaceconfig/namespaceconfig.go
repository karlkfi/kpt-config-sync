package namespaceconfig

import (
	"context"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/status"
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
	if err := cache.List(ctx, namespaceConfigs); err != nil {
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
	configs.Syncs, err = listSyncs(ctx, cache)
	return &configs, err
}

// DecorateWithClusterConfigs updates AllPolices with all the ClusterConfigs from APIServer.
func DecorateWithClusterConfigs(ctx context.Context, reader client.Reader, policies *AllConfigs) status.MultiError {
	clusterConfigs := &v1.ClusterConfigList{}
	if err := reader.List(ctx, clusterConfigs); err != nil {
		return status.APIServerError(err, "failed to list ClusterConfigs")
	}

	for _, c := range clusterConfigs.Items {
		switch n := c.Name; n {
		case v1.ClusterConfigName:
			policies.ClusterConfig = c.DeepCopy()
		case v1.CRDClusterConfigName:
			policies.CRDClusterConfig = c.DeepCopy()
		default:
			return status.UndocumentedErrorf("found an invalid ClusterConfig: %s", n)
		}
	}
	return nil
}

// listSyncs gets a map-by-name of Syncs currently present in the cluster from
// the cache.
func listSyncs(ctx context.Context, cache cache.Cache) (map[string]v1.Sync, error) {
	syncs := &v1.SyncList{}
	if err := cache.List(ctx, syncs); err != nil {
		return nil, errors.Wrap(err, "failed to list Syncs")
	}

	ret := make(map[string]v1.Sync, len(syncs.Items))
	for _, s := range syncs.Items {
		ret[s.Name] = *s.DeepCopy()
	}
	return ret, nil
}
