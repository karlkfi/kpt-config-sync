package parse

import (
	"context"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/importer"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/namespaceconfig"
)

const timeout = time.Second * 15

// NewParser constructs a Parser from ParserOpt.
func NewParser(parserOpt filesystem.ParserOpt) (*filesystem.Parser, error) {
	if parserOpt.RootPath.Equal(filesystem.ParserOpt{}.RootPath) {
		return nil, status.InternalError("No root path specified.")
	}

	return filesystem.NewParser(importer.DefaultCLIOptions, parserOpt), nil
}

// Parse parses a GKE Policy Directory with a Parser using the specified Parser optional arguments.
// Exits early if it encounters parsing/validation errors.
func Parse(clusterName string, parserOpt filesystem.ParserOpt) (*namespaceconfig.AllConfigs, error) {
	p, err := NewParser(parserOpt)
	if err != nil {
		return nil, err
	}

	config, err := restconfig.NewRestConfig()
	if err != nil {
		glog.Fatalf("Failed to create rest config: %+v", err)
	}

	if err := filesystem.ValidateInstallation(importer.DefaultCLIOptions); err != nil {
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
