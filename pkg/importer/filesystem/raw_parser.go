package filesystem

import (
	"fmt"
	"time"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	utildiscovery "github.com/google/nomos/pkg/util/discovery"
	"github.com/google/nomos/pkg/util/namespaceconfig"
)

// RawParser parses a directory of raw YAML resource manifests into an AllConfigs usable by the
// syncer.
//
// This currently lacks much of the validation of Parser, but we may decide to add it in later.
// TODO(b/137202024)
type RawParser struct {
	path         cmpath.Relative
	reader       Reader
	clientGetter genericclioptions.RESTClientGetter
}

var _ configParser = &RawParser{}

// NewRawParser instantiates a RawParser.
func NewRawParser(path cmpath.Relative, reader Reader, client genericclioptions.RESTClientGetter) *RawParser {
	return &RawParser{
		path:         path,
		reader:       reader,
		clientGetter: client,
	}
}

// Parse reads a directory of raw, unstructured YAML manifests and outputs the resulting AllConfigs.
func (p *RawParser) Parse(importToken string, currentConfigs *namespaceconfig.AllConfigs, loadTime time.Time, _ string) (*namespaceconfig.AllConfigs, status.MultiError) {
	// Get all known API resources from the server.
	dc, err := p.clientGetter.ToDiscoveryClient()
	if err != nil {
		return nil, status.From(status.APIServerWrapf(err, "failed to get discovery client"))
	}
	apiResources, err := dc.ServerResources()
	if err != nil {
		return nil, status.From(status.APIServerWrapf(err, "failed to get server resources"))
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
		return nil, status.From(err)
	}
	scoper.AddCustomResources(crds...)

	// Read all manifests and extract them into FileObjects.
	fileObjects, errs := p.reader.Read(p.path, false, crds...)
	if errs != nil {
		return nil, errs
	}
	fileObjects = deduplicate(fileObjects)
	result := namespaceconfig.NewAllConfigs(importToken, loadTime)
	for _, f := range fileObjects {
		if f.GroupVersionKind() == kinds.Namespace() {
			// Namespace is a snowflake.
			// This preserves the ordering behavior of kubectl apply -f. This means what is in the
			// alphabetically-last file wins.
			result.AddNamespaceConfig(f.Name(), f.MetaObject().GetAnnotations(), f.MetaObject().GetLabels())
			continue
		}

		result.AddSync(*v1.NewSync(f.GroupVersionKind().GroupKind()))
		switch scoper.GetScope(f.GroupVersionKind().GroupKind()) {
		case utildiscovery.ClusterScope:
			result.AddClusterResource(f.Object)
		case utildiscovery.NamespaceScope:
			namespace := f.Namespace()
			if namespace == "" {
				// Empty string/non-declared metadata.namespace automatically maps to "default", so this
				// ensures we maintain these in a single NamespaceConfig entry.
				namespace = "default"
			}
			result.AddNamespaceResource(namespace, f.Object)
		case utildiscovery.UnknownScope:
			errs = status.Append(errs, vet.UnknownObjectError(&f))
		}
	}

	return result, errs
}

func deduplicate(os []ast.FileObject) []ast.FileObject {
	// First object found wins. Others are ignored.
	m := map[string]bool{}
	var result []ast.FileObject

	for _, o := range os {
		id := fmt.Sprintf("%s/%s/%s", o.GroupVersionKind().String(), o.Namespace(), o.Name())
		if m[id] {
			continue
		}
		m[id] = true
		result = append(result, o)
	}
	return result
}
