package parse

import (
	"context"
	"time"

	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"github.com/pkg/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const timeout = time.Second * 15

// Parse parses a GKE Policy Directory with a Parser using the specified Parser optional arguments.
// Exits early if it encounters parsing/validation errors.
func Parse(clusterName string, parserOpt filesystem.ParserOpt) (*namespaceconfig.AllConfigs, error) {
	if parserOpt.RootPath.Equal(filesystem.ParserOpt{}.RootPath) {
		return nil, status.InternalError.New("No root path specified.")
	}

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

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	policies, cErr := clusterConfigs(ctx, config)
	if cErr != nil {
		return nil, cErr
	}
	resources, mErr := p.Parse("", policies, time.Time{}, clusterName)
	if mErr != nil {
		return nil, errors.Wrap(mErr, "Found issues")
	}

	return resources, nil
}

// clusterConfigs returns an AllPolicies with only the ClusterConfigs populated.
func clusterConfigs(ctx context.Context, config *rest.Config) (*namespaceconfig.AllConfigs, error) {
	mapper, err := apiutil.NewDiscoveryRESTMapper(config)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create mapper")
	}

	s := runtime.NewScheme()
	if sErr := v1.AddToScheme(s); sErr != nil {
		return nil, errors.Wrap(sErr, "could not add configmanagement types to scheme")
	}
	c, cErr := client.New(config, client.Options{
		Scheme: s,
		Mapper: mapper,
	})
	if cErr != nil {
		return nil, errors.Wrapf(cErr, "failed to create client")
	}
	configs := &namespaceconfig.AllConfigs{}
	err = namespaceconfig.DecorateWithClusterConfigs(ctx, c, configs)
	return configs, err
}
