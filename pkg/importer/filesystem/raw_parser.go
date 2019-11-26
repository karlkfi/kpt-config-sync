package filesystem

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes/scheme"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/decode"
	"github.com/google/nomos/pkg/util/clusterconfig"
	utildiscovery "github.com/google/nomos/pkg/util/discovery"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RawParser parses a directory of raw YAML resource manifests into an AllConfigs usable by the
// syncer.
type RawParser struct {
	path         cmpath.Relative
	reader       Reader
	clientGetter genericclioptions.RESTClientGetter
}

var _ ConfigParser = &RawParser{}

// NewRawParser instantiates a RawParser.
func NewRawParser(path cmpath.Relative, reader Reader, client genericclioptions.RESTClientGetter) *RawParser {
	return &RawParser{
		path:         path,
		reader:       reader,
		clientGetter: client,
	}
}

// Parse reads a directory of raw, unstructured YAML manifests and outputs the resulting AllConfigs.
func (p *RawParser) Parse(importToken string, currentConfigs *namespaceconfig.AllConfigs, loadTime metav1.Time, _ string) (*namespaceconfig.AllConfigs, status.MultiError) {
	// Get all known API resources from the server.
	dc, err := p.clientGetter.ToDiscoveryClient()
	if err != nil {
		return nil, status.APIServerError(err, "failed to get discovery client")
	}
	apiResources, err := dc.ServerResources()
	if err != nil {
		return nil, status.APIServerError(err, "failed to get server resources")
	}
	apiInfo, err := utildiscovery.NewAPIInfo(apiResources)
	if err != nil {
		return nil, status.APIServerError(err, "discovery failed for server resources")
	}

	// Read any CRDs in the directory so the parser is aware of them.
	crds, errs := readCRDs(p.reader, p.path)
	if errs != nil {
		return nil, errs
	}

	// Combine server-side API resources and declared CRDs into the scoper that can determine whether
	// an object is namespace or cluster scoped.
	scoper, err := utildiscovery.NewAPIInfo(apiResources)
	if err != nil {
		return nil, status.APIServerError(err, "error getting APIResources from Kubernetes cluster")
	}
	scoper.AddCustomResources(crds...)

	// Read all manifests and extract them into FileObjects.
	fileObjects, errs := p.reader.Read(p.path, false, crds...)
	if errs != nil {
		return nil, errs
	}

	var crdErr status.Error
	crdInfo, crdErr := clusterconfig.NewCRDInfo(
		decode.NewGenericResourceDecoder(scheme.Scheme),
		&v1.ClusterConfig{},
		crds)
	if crdErr != nil {
		return nil, crdErr
	}

	errs = status.Append(errs, standardValidation(fileObjects))

	var validators = []nonhierarchical.Validator{
		nonhierarchical.IllegalHierarchicalKindValidator,
		nonhierarchical.CRDRemovalValidator(crdInfo),
		nonhierarchical.ScopeValidator(apiInfo),
	}
	for _, v := range validators {
		errs = status.Append(errs, v.Validate(fileObjects))
	}
	if errs != nil {
		return nil, errs
	}

	return namespaceconfig.NewAllConfigs(importToken, loadTime, apiInfo, fileObjects)
}

// ReadClusterRegistryResources returns empty as Cluster declarations are forbidden if hierarchical
// parsing is disabled.
func (p *RawParser) ReadClusterRegistryResources() []ast.FileObject {
	return nil
}
