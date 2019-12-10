package filesystem

import (
	"github.com/google/nomos/pkg/importer/customresources"
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
	clientGetter utildiscovery.ClientGetter
}

var _ ConfigParser = &RawParser{}

// NewRawParser instantiates a RawParser.
func NewRawParser(path cmpath.Relative, reader Reader, client utildiscovery.ClientGetter) *RawParser {
	return &RawParser{
		path:         path,
		reader:       reader,
		clientGetter: client,
	}
}

// Parse reads a directory of raw, unstructured YAML manifests and outputs the resulting AllConfigs.
func (p *RawParser) Parse(importToken string, currentConfigs *namespaceconfig.AllConfigs, loadTime metav1.Time, _ string) (*namespaceconfig.AllConfigs, status.MultiError) {
	// Read all manifests and extract them into FileObjects.
	fileObjects, errs := p.reader.Read(p.path)
	if errs != nil {
		return nil, errs
	}

	crds, crdErrs := customresources.GetCRDs(fileObjects)
	if crdErrs != nil {
		errs = status.Append(errs, crdErrs)
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

	// Get all known API resources from the server.
	apiResources, err := utildiscovery.GetResourcesFromClientGetter(p.clientGetter)
	if err != nil {
		return nil, err
	}
	scoper, err := utildiscovery.NewScoperFromServerResources(apiResources)
	if err != nil {
		return nil, status.APIServerError(err, "discovery failed for server resources")
	}

	// Combine server-side API resources and declared CRDs into the scoper that can determine whether
	// an object is namespace or cluster scoped.
	scoper.AddCustomResources(crds)
	var validators = []nonhierarchical.Validator{
		nonhierarchical.IllegalHierarchicalKindValidator,
		nonhierarchical.CRDRemovalValidator(crdInfo),
		nonhierarchical.ScopeValidator(scoper),
	}
	for _, v := range validators {
		errs = status.Append(errs, v.Validate(fileObjects))
	}
	if errs != nil {
		return nil, errs
	}

	return namespaceconfig.NewAllConfigs(importToken, loadTime, scoper, fileObjects)
}

// ReadClusterRegistryResources returns empty as Cluster declarations are forbidden if hierarchical
// parsing is disabled.
func (p *RawParser) ReadClusterRegistryResources() []ast.FileObject {
	return nil
}
