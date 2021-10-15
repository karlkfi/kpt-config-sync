package declared

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/nomos/pkg/testing/openapitest"
	openapiv2 "github.com/googleapis/gnostic/openapiv2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/kube-openapi/pkg/schemaconv"
	"k8s.io/kube-openapi/pkg/util/proto"
	"k8s.io/kubectl/pkg/util/openapi"
	"sigs.k8s.io/structured-merge-diff/v4/typed"
)

// ValueConverter converts a runtime.Object into a TypedValue.
type ValueConverter struct {
	discoveryClient  discovery.DiscoveryInterface
	openAPIResources openapi.Resources
	parser           *typed.Parser
}

// NewValueConverter returns a ValueConverter initialized with the given
// discovery client.
func NewValueConverter(dc discovery.DiscoveryInterface) (*ValueConverter, error) {
	v := &ValueConverter{discoveryClient: dc}
	if err := v.Refresh(); err != nil {
		return nil, err
	}
	return v, nil
}

// Refresh pulls fresh schemas from the openapi discovery endpoint and
// instantiates the ValueConverter with them. This can be called periodically as
// new custom types (eg CRDs) are added to the cluster.
func (v *ValueConverter) Refresh() error {
	oa, err := openapi.NewOpenAPIParser(v.discoveryClient).Parse()
	if err != nil {
		return err
	}
	doc, err := v.discoveryClient.OpenAPISchema()
	if err != nil {
		return err
	}
	parser, err := typedParser(doc)
	if err != nil {
		return err
	}
	v.openAPIResources = oa
	v.parser = parser
	return nil
}

// TypedValue returns the equivalent TypedValue for the given Object.
func (v *ValueConverter) TypedValue(obj runtime.Object) (*typed.TypedValue, error) {
	gvk := obj.GetObjectKind().GroupVersionKind()
	res := v.openAPIResources.LookupResource(gvk)
	if res == nil {
		// TODO(b/160786679): We probably want to Refresh at this point? Can we be
		// proactive about watching for new CRDs and refreshing when they are become
		// established?
		return typedValueDeduced(obj)
	}

	t := v.parser.Type(res.GetPath().String())
	switch o := obj.(type) {
	case *unstructured.Unstructured:
		return t.FromUnstructured(o.UnstructuredContent())
	default:
		return t.FromStructured(obj)
	}
}

func typedValueDeduced(obj runtime.Object) (*typed.TypedValue, error) {
	switch o := obj.(type) {
	case *unstructured.Unstructured:
		return typed.DeducedParseableType.FromUnstructured(o.UnstructuredContent())
	default:
		return typed.DeducedParseableType.FromStructured(obj)
	}
}

// typedParser returns a typed.Parser instantiated with schemas from the given
// openapi Document.
func typedParser(doc *openapiv2.Document) (*typed.Parser, error) {
	models, err := proto.NewOpenAPIData(doc)
	if err != nil {
		return nil, fmt.Errorf("interpreting models: %w", err)
	}
	typeSchema, err := schemaconv.ToSchemaWithPreserveUnknownFields(models, false)
	if err != nil {
		return nil, fmt.Errorf("converting models to schema: %w", err)
	}
	// This fails the copylocks lint check, but this is exactly how the API server
	// does it so I'm not sure what else to do.
	return &typed.Parser{Schema: *typeSchema}, nil //nolint:govet
}

// ValueConverterForTest returns a ValueConverter initialized for unit tests.
func ValueConverterForTest() (*ValueConverter, error) {
	doc, err := openapitest.Doc(pathToTestFile())
	if err != nil {
		return nil, err
	}
	oa, err := openapi.NewOpenAPIData(doc)
	if err != nil {
		return nil, err
	}

	parser, err := typedParser(doc)
	if err != nil {
		return nil, err
	}

	return &ValueConverter{nil, oa, parser}, nil
}

func pathToTestFile() string {
	path, err := os.Getwd()
	if err != nil {
		return err.Error()
	}

	// All of our code sits in subdirectories (pkg, cmd, etc) under this path:
	// github.com/google/nomos/...
	// Some unit tests run from pkg and some from cmd, so we climb up from the
	// working directory to "google", and then specify back down to the file. We
	// can't use "nomos" because that directory name is repeated in:
	// github.com/google/nomos/cmd/nomos
	for filepath.Base(path) != "google" {
		path = filepath.Dir(path)
	}
	return filepath.Join(path, "nomos", "pkg", "testing", "openapitest", "openapi_v2.txt")
}
