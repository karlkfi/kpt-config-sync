package parse

import (
	"context"
	"time"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/importer"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/decode"
	"github.com/google/nomos/pkg/util/clusterconfig"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"github.com/pkg/errors"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

const timeout = time.Second * 15

// NewParser returns a new default-initialized Parser for the CLI.
func NewParser() *filesystem.Parser {
	return filesystem.NewParser(&filesystem.FileReader{}, importer.DefaultCLIOptions)
}

// Parse parses a GKE Policy Directory with a Parser using the specified Parser optional arguments.
// Exits early if it encounters parsing/validation errors.
func Parse(clusterName string, root cmpath.Absolute, enableAPIServerChecks bool) (*namespaceconfig.AllConfigs, error) {
	p := NewParser()

	if err := filesystem.ValidateInstallation(importer.DefaultCLIOptions); err != nil {
		return nil, errors.Wrap(err, "Found issues")
	}

	trackedFiles, err := FindFiles(root)
	if err != nil {
		return nil, err
	}

	fileObjects, mErr := p.Parse(clusterName, enableAPIServerChecks, GetSyncedCRDs,
		root, filesystem.FilterHierarchyFiles(root, trackedFiles))
	if mErr != nil {
		return nil, errors.Wrap(mErr, "Found issues")
	}

	return namespaceconfig.NewAllConfigs("", metav1.Time{}, fileObjects), nil
}

// GetSyncedCRDs returns the CRDs synced to the cluster in the current context.
//
// Times out after 15 seconds.
func GetSyncedCRDs() ([]*v1beta1.CustomResourceDefinition, status.MultiError) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	config, err := restconfig.NewRestConfig()
	if err != nil {
		return nil, getSyncedCRDsError(err, "failed to create rest config: %+v")
	}

	mapper, err := apiutil.NewDiscoveryRESTMapper(config)
	if err != nil {
		return nil, getSyncedCRDsError(err, "failed to create mapper")
	}

	s := runtime.NewScheme()
	if sErr := v1.AddToScheme(s); sErr != nil {
		return nil, getSyncedCRDsError(sErr, "could not add configmanagement types to scheme")
	}
	c, cErr := client.New(config, client.Options{
		Scheme: s,
		Mapper: mapper,
	})
	if cErr != nil {
		return nil, getSyncedCRDsError(cErr, "failed to create client")
	}
	configs := &namespaceconfig.AllConfigs{}
	decorateErr := namespaceconfig.DecorateWithClusterConfigs(ctx, c, configs)
	if decorateErr != nil {
		return nil, decorateErr
	}

	decoder := decode.NewGenericResourceDecoder(scheme.Scheme)
	syncedCRDs, crdErr := clusterconfig.GetCRDs(decoder, configs.ClusterConfig)
	if crdErr != nil {
		// We were unable to parse the CRDs from the current ClusterConfig, so bail out.
		// TODO(b/146139870): Make error message more user-friendly when this happens.
		return nil, crdErr
	}
	return syncedCRDs, nil
}

func getSyncedCRDsError(err error, message string) status.Error {
	return status.APIServerError(err, message+". Did you mean to run with --no-api-server-check?")
}
