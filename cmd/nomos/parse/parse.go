package parse

import (
	"fmt"
	"time"

	"github.com/google/nomos/pkg/api/configmanagement/v1"

	"github.com/golang/glog"
	"github.com/google/nomos/clientgen/informer"
	"github.com/google/nomos/pkg/client/meta"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/service"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"github.com/pkg/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
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
	p := filesystem.NewParser(&genericclioptions.ConfigFlags{}, parserOpt)
	if err := p.ValidateInstallation(); err != nil {
		return nil, errors.Wrap(err, "Found issues")
	}

	stopCh := make(chan struct{})
	go service.WaitForShutdownSignalCb(stopCh)

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
	client, err := meta.NewForConfig(config, time.Minute)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create client")
	}

	informerFactory := informer.NewSharedInformerFactory(
		client.ConfigManagement(), time.Minute)
	_, iErr := informerFactory.ForResource(v1.SchemeGroupVersion.WithResource("clusterconfigs"))
	if iErr != nil {
		return nil, errors.Wrap(iErr, "failed get clusterconfig informater")
	}
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
