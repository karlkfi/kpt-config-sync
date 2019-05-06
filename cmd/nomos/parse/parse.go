package parse

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/clientgen/informer"
	"github.com/google/nomos/pkg/client/meta"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/importer"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"github.com/pkg/errors"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

// Parse parses a GKE Policy Directory with a Parser using the specified Parser optional arguments.
// Exits early if it encounters parsing/validation errors.
func Parse(dir string, parserOpt filesystem.ParserOpt) (*namespaceconfig.AllConfigs, error) {
	config, err := restconfig.NewRestConfig()
	if err != nil {
		glog.Fatalf("Failed to create rest config: %+v", err)
	}

	// TODO(119066037): Override the host in a way that doesn't involve overwriting defaults set internally in client-go.
	clientcmd.ClusterDefaults = clientcmdapi.Cluster{Server: config.Host}
	factoryFactory := func(crds ...*v1beta1.CustomResourceDefinition) cmdutil.Factory {
		return cmdutil.NewFactory(importer.NewFilesystemCRDAwareClientGetter(&genericclioptions.ConfigFlags{}, crds...))
	}
	p, err := filesystem.NewParser(factoryFactory, parserOpt)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create parser")
	}

	stopCh := make(chan struct{})
	policies, cErr := clusterConfigs(config, stopCh)
	if cErr != nil {
		return nil, cErr
	}
	resources, mErr := p.Parse(dir, "", policies, time.Time{})
	if mErr != nil {
		return nil, errors.Wrap(mErr, "Found issues")
	}

	return resources, nil
}

// clusterConfigs returns an AllPolicies with only the ClusterConfigs populated.
func clusterConfigs(config *rest.Config, stopCh <-chan struct{}) (*namespaceconfig.AllConfigs, error) {
	minute := time.Minute
	client, err := meta.NewForConfig(config, &minute)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create client")
	}

	informerFactory := informer.NewSharedInformerFactory(
		client.ConfigManagement(), minute)
	informerFactory.Start(stopCh)
	synced := informerFactory.WaitForCacheSync(stopCh)
	for syncType, ok := range synced {
		if !ok {
			elemType := syncType.Elem()
			return nil, fmt.Errorf("failed to sync %s:%s", elemType.PkgPath(), elemType.Name())
		}
	}
	lister := informerFactory.Configmanagement().V1().ClusterConfigs().Lister()

	configs := &namespaceconfig.AllConfigs{}
	err = namespaceconfig.DecorateWithClusterConfigs(lister, configs)
	return configs, err
}
