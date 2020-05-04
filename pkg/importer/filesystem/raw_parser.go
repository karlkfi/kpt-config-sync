package filesystem

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/transform"
	"github.com/google/nomos/pkg/importer/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/importer/analyzer/validation"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/importer/customresources"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
	utildiscovery "github.com/google/nomos/pkg/util/discovery"
)

// RawParser parses a directory of raw YAML resource manifests into an AllConfigs usable by the
// syncer.
type rawParser struct {
	reader       Reader
	clientGetter utildiscovery.ClientGetter
}

var _ ConfigParser = &rawParser{}

// NewRawParser instantiates a RawParser.
func NewRawParser(reader Reader, client utildiscovery.ClientGetter) ConfigParser {
	return &rawParser{
		reader:       reader,
		clientGetter: client,
	}
}

// Parse reads a directory of raw, unstructured YAML manifests and outputs the resulting AllConfigs.
func (p *rawParser) Parse(
	clusterName string,
	enableAPIServerChecks bool,
	getSyncedCRDs GetSyncedCRDs,
	policyDir cmpath.Absolute,
	files []cmpath.Absolute,
) ([]ast.FileObject, status.MultiError) {
	// Read all manifests and extract them into FileObjects.
	fileObjects, errs := p.reader.Read(policyDir, files)
	if errs != nil {
		return nil, errs
	}

	declaredCRDs, crdErrs := customresources.GetCRDs(fileObjects)
	if crdErrs != nil {
		return nil, crdErrs
	}

	scoper, syncedCRDs, scoperErr := buildScoper(p.clientGetter, enableAPIServerChecks, fileObjects, declaredCRDs, getSyncedCRDs)
	if scoperErr != nil {
		return nil, scoperErr
	}

	scopeErrs := nonhierarchical.ScopeValidator(scoper).Validate(fileObjects)
	if scopeErrs != nil {
		// Don't try to resolve selectors if scopes are incorrect.
		return nil, scopeErrs
	}
	fileObjects, selErr := resolveFlatSelectors(scoper, clusterName, fileObjects)
	if selErr != nil {
		return nil, selErr
	}

	errs = status.Append(errs, standardValidation(fileObjects))

	var validators = []nonhierarchical.Validator{
		nonhierarchical.IllegalHierarchicalKindValidator,
		nonhierarchical.CRDRemovalValidator(syncedCRDs, declaredCRDs),
	}
	for _, v := range validators {
		errs = status.Append(errs, v.Validate(fileObjects))
	}
	return fileObjects, errs
}

// ReadClusterRegistryResources returns empty as Cluster declarations are forbidden if hierarchical
// parsing is disabled.
func (p *rawParser) ReadClusterRegistryResources(root cmpath.Absolute, files []cmpath.Absolute) []ast.FileObject {
	return nil
}

func resolveFlatSelectors(scoper utildiscovery.Scoper, clusterName string, fileObjects []ast.FileObject) ([]ast.FileObject, status.MultiError) {
	annErr := nonhierarchical.NewSelectorAnnotationValidator(scoper, true).Validate(fileObjects)
	if annErr != nil {
		return nil, annErr
	}

	fileObjects = nonhierarchical.CopyAbstractResources(fileObjects)

	csuErr := validation.ClusterSelectorUniqueness.Validate(fileObjects)
	if csuErr != nil {
		return nil, csuErr
	}

	fileObjects, csErr := selectors.ResolveClusterSelectors(clusterName, fileObjects)
	if csErr != nil {
		return nil, csErr
	}

	nsuErr := validation.NamespaceSelectorUniqueness.Validate(fileObjects)
	if nsuErr != nil {
		return nil, nsuErr
	}

	fileObjects, nsErr := selectors.ResolveFlatNamespaceSelectors(fileObjects)
	if nsErr != nil {
		return nil, nsErr
	}

	return transform.RemoveEphemeralResources(fileObjects), nil
}
