package tree

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/customresources"
	"github.com/google/nomos/pkg/status"
	utildiscovery "github.com/google/nomos/pkg/util/discovery"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
)

// APIInfoBuilderVisitor adds APIInfo to Extensions
type APIInfoBuilderVisitor struct {
	*visitor.Base
	client             discovery.ServerResourcesInterface
	ephemeralResources []*metav1.APIResourceList
	errs               status.MultiError
}

// NewAPIInfoBuilderVisitor instantiates a CRDClusterConfigInfoVisitor with a set of objects to add.
func NewAPIInfoBuilderVisitor(client discovery.ServerResourcesInterface, ephemeralResources []*metav1.APIResourceList) *APIInfoBuilderVisitor {
	v := &APIInfoBuilderVisitor{
		Base:               visitor.NewBase(),
		client:             client,
		ephemeralResources: ephemeralResources,
	}
	v.SetImpl(v)
	return v
}

// VisitRoot adds APIInfo to Root Extensions.
func (v *APIInfoBuilderVisitor) VisitRoot(r *ast.Root) *ast.Root {
	resources, discoveryErr := v.client.ServerResources()
	if discoveryErr != nil {
		v.errs = status.Append(v.errs, status.APIServerError(discoveryErr, "failed to get server resources"))
	}

	resources = append(resources, v.ephemeralResources...)
	scoper, err := utildiscovery.NewScoperFromServerResources(resources)
	if err != nil {
		v.errs = status.Append(v.errs, status.APIServerError(err, "failed to parse server resources"))
		return r
	}

	crds, errs := customresources.ProcessClusterObjects(r.ClusterObjects)
	if errs != nil {
		v.errs = status.Append(v.errs, errs)
	}
	scoper.AddCustomResources(crds...)

	v.errs = status.Append(v.errs, utildiscovery.AddScoper(r, scoper))
	return r
}

// Error implements Visitor.
func (v *APIInfoBuilderVisitor) Error() status.MultiError {
	return v.errs
}
