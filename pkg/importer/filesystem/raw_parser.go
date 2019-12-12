package filesystem

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/importer/customresources"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
	utildiscovery "github.com/google/nomos/pkg/util/discovery"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

// RawParser parses a directory of raw YAML resource manifests into an AllConfigs usable by the
// syncer.
type RawParser struct {
	root         cmpath.Root
	reader       Reader
	clientGetter utildiscovery.ClientGetter
}

var _ ConfigParser = &RawParser{}

// NewRawParser instantiates a RawParser.
func NewRawParser(path cmpath.Root, reader Reader, client utildiscovery.ClientGetter) *RawParser {
	return &RawParser{
		root:         path,
		reader:       reader,
		clientGetter: client,
	}
}

// Parse reads a directory of raw, unstructured YAML manifests and outputs the resulting AllConfigs.
func (p *RawParser) Parse(syncedCRDs []*v1beta1.CustomResourceDefinition, _ string) ([]ast.FileObject, status.MultiError) {
	// Read all manifests and extract them into FileObjects.
	fileObjects, errs := p.reader.Read(p.root)
	if errs != nil {
		return nil, errs
	}

	declaredCrds, crdErrs := customresources.GetCRDs(fileObjects)
	if crdErrs != nil {
		errs = status.Append(errs, crdErrs)
		return nil, errs
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
	scoper.AddCustomResources(declaredCrds)

	var validators = []nonhierarchical.Validator{
		nonhierarchical.IllegalHierarchicalKindValidator,
		nonhierarchical.CRDRemovalValidator(syncedCRDs, declaredCrds),
		nonhierarchical.ScopeValidator(scoper),
	}
	for _, v := range validators {
		errs = status.Append(errs, v.Validate(fileObjects))
	}
	return fileObjects, errs
}

// ReadClusterRegistryResources returns empty as Cluster declarations are forbidden if hierarchical
// parsing is disabled.
func (p *RawParser) ReadClusterRegistryResources() []ast.FileObject {
	return nil
}
