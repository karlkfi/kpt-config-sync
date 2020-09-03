package filesystem

import (
	"github.com/google/nomos/pkg/core"
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
	reader Reader
	dc     utildiscovery.ServerResourcer
}

var _ ConfigParser = &rawParser{}

// NewRawParser instantiates a RawParser.
func NewRawParser(reader Reader, dc utildiscovery.ServerResourcer) ConfigParser {
	return &rawParser{
		reader: reader,
		dc:     dc,
	}
}

// Parse reads a directory of raw, unstructured YAML manifests and outputs the resulting AllConfigs.
func (p *rawParser) Parse(
	clusterName string,
	enableAPIServerChecks bool,
	getSyncedCRDs GetSyncedCRDs,
	policyDir cmpath.Absolute,
	files []cmpath.Absolute,
) ([]core.Object, status.MultiError) {
	// Read all manifests and extract them into FileObjects.
	fileObjects, errs := p.reader.Read(policyDir, files)
	if errs != nil {
		return nil, errs
	}

	declaredCRDs, crdErrs := customresources.GetCRDs(fileObjects)
	if crdErrs != nil {
		return nil, crdErrs
	}

	scoper, syncedCRDs, scoperErr := BuildScoper(p.dc, enableAPIServerChecks, fileObjects, declaredCRDs, getSyncedCRDs)
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
	return AsCoreObjects(fileObjects), errs
}

// ReadClusterRegistryResources returns empty as Cluster declarations are forbidden if hierarchical
// parsing is disabled.
func (p *rawParser) ReadClusterRegistryResources(_ cmpath.Absolute, _ []cmpath.Absolute) []ast.FileObject {
	return nil
}

func resolveFlatSelectors(scoper utildiscovery.Scoper, clusterName string, fileObjects []ast.FileObject) ([]ast.FileObject, status.MultiError) {
	// Validate and resolve cluster selectors.
	err := nonhierarchical.NewClusterSelectorAnnotationValidator().Validate(fileObjects)
	if err != nil {
		return nil, err
	}

	fileObjects = nonhierarchical.CopyAbstractResources(fileObjects)

	err = validation.ClusterSelectorUniqueness.Validate(fileObjects)
	if err != nil {
		return nil, err
	}

	fileObjects, err = selectors.ResolveClusterSelectors(clusterName, fileObjects)
	if err != nil {
		return nil, err
	}

	// Validate and resolve namespace selectors.
	err = nonhierarchical.NewNamespaceSelectorAnnotationValidator(scoper, true).Validate(fileObjects)
	if err != nil {
		return nil, err
	}

	err = validation.NamespaceSelectorUniqueness.Validate(fileObjects)
	if err != nil {
		return nil, err
	}

	fileObjects, err = selectors.ResolveFlatNamespaceSelectors(fileObjects)
	if err != nil {
		return nil, err
	}

	return transform.RemoveEphemeralResources(fileObjects), nil
}
