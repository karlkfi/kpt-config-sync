package filesystem

import (
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/transform"
	"github.com/google/nomos/pkg/importer/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/importer/analyzer/validation"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/importer/customresources"
	"github.com/google/nomos/pkg/importer/reader"
	"github.com/google/nomos/pkg/status"
	utildiscovery "github.com/google/nomos/pkg/util/discovery"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

// RawParser parses a directory of raw YAML resource manifests into an AllConfigs usable by the
// syncer.
type rawParser struct {
	reader                reader.Reader
	errOnUnknownKinds     bool
	defaultNamespace      string
	inNamespaceReconciler bool
}

var _ ConfigParser = &rawParser{}

// NewRawParser instantiates a RawParser.
func NewRawParser(reader reader.Reader, errOnUnknownKinds bool, defaultNamespace string, reconcilerScope declared.Scope) ConfigParser {
	return &rawParser{
		reader:                reader,
		errOnUnknownKinds:     errOnUnknownKinds,
		defaultNamespace:      defaultNamespace,
		inNamespaceReconciler: reconcilerScope != declared.RootReconciler,
	}
}

// Parse reads a directory of raw, unstructured YAML manifests and outputs the resulting AllConfigs.
func (p *rawParser) Parse(clusterName string, syncedCRDs []*v1beta1.CustomResourceDefinition, buildScoper utildiscovery.BuildScoperFunc, filePaths reader.FilePaths) ([]ast.FileObject, status.MultiError) {
	// Read all manifests and extract them into FileObjects.
	fileObjects, errs := p.reader.Read(filePaths)
	if errs != nil {
		return nil, errs
	}

	declaredCRDs, crdErrs := customresources.GetCRDs(fileObjects)
	if crdErrs != nil {
		return nil, crdErrs
	}

	scoper, scoperErr := buildScoper(declaredCRDs, fileObjects)
	if scoperErr != nil {
		return nil, scoperErr
	}

	scopeErrs := nonhierarchical.ScopeValidator(p.inNamespaceReconciler, p.defaultNamespace, scoper, p.errOnUnknownKinds).Validate(fileObjects)
	if scopeErrs != nil {
		// Don't try to resolve selectors if scopes are incorrect.
		return nil, scopeErrs
	}
	fileObjects, selErr := resolveFlatSelectors(scoper, clusterName, fileObjects, p.errOnUnknownKinds)
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

	fileObjects = selectors.AnnotateClusterName(clusterName, fileObjects)
	return fileObjects, errs
}

// ReadClusterRegistryResources returns empty as Cluster declarations are forbidden if hierarchical
// parsing is disabled.
func (p *rawParser) ReadClusterRegistryResources(_ reader.FilePaths) []ast.FileObject {
	return nil
}

func resolveFlatSelectors(scoper utildiscovery.Scoper, clusterName string, fileObjects []ast.FileObject, enableAPIServerChecks bool) ([]ast.FileObject, status.MultiError) {
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
	err = nonhierarchical.NewNamespaceSelectorAnnotationValidator(scoper, enableAPIServerChecks).Validate(fileObjects)
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
