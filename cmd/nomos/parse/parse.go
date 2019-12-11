package parse

import (
	"context"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/importer"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

const timeout = time.Second * 15

// NewParser constructs a Parser from ParserOpt.
func NewParser(root cmpath.Root) *filesystem.Parser {
	return filesystem.NewParser(root, &filesystem.FileReader{}, importer.DefaultCLIOptions)
}

// Parse parses a GKE Policy Directory with a Parser using the specified Parser optional arguments.
// Exits early if it encounters parsing/validation errors.
func Parse(clusterName string, root cmpath.Root) (*namespaceconfig.AllConfigs, error) {
	p := NewParser(root)

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
	resources, mErr := p.Parse("", policies, metav1.Time{}, clusterName)
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
