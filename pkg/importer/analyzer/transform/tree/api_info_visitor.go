package tree

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	utildiscovery "github.com/google/nomos/pkg/util/discovery"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
)

// APIInfoBuilderVisitor adds APIInfo to Extensions
type APIInfoBuilderVisitor struct {
	*visitor.Base
	client             discovery.ServerResourcesInterface
	ephemeralResources []*metav1.APIResourceList
	enableCRDs         bool
	errs               status.MultiError
}

// NewAPIInfoBuilderVisitor instantiates an CRDClusterConfigInfoVisitor with a set of objects to add.
func NewAPIInfoBuilderVisitor(client discovery.ServerResourcesInterface, ephemeralResources []*metav1.APIResourceList,
	enableCRDs bool) *APIInfoBuilderVisitor {
	v := &APIInfoBuilderVisitor{
		Base:               visitor.NewBase(),
		client:             client,
		ephemeralResources: ephemeralResources,
		enableCRDs:         enableCRDs,
	}
	v.SetImpl(v)
	return v
}

// VisitRoot adds APIInfo to Root Extensions.
func (v *APIInfoBuilderVisitor) VisitRoot(r *ast.Root) *ast.Root {
	resources, discoveryErr := v.client.ServerResources()
	if discoveryErr != nil {
		v.errs = status.Append(v.errs, status.APIServerWrapf(discoveryErr, "failed to get server resources"))
	}

	resources = append(resources, v.ephemeralResources...)
	apiInfo, err := utildiscovery.NewAPIInfo(resources)
	if err != nil {
		v.errs = status.Append(v.errs, status.APIServerWrapf(err, "discovery failed for server resources"))
	}
	if v.enableCRDs {
		for _, cr := range r.ClusterObjects {
			if cr.GroupVersionKind() != kinds.CustomResourceDefinition() {
				continue
			}
			apiInfo.AddCustomResources(cr.Object.(*v1beta1.CustomResourceDefinition))
		}
	}
	v.errs = status.Append(v.errs, utildiscovery.AddAPIInfo(r, apiInfo))

	return r
}

// Error implements Visitor.
func (v *APIInfoBuilderVisitor) Error() status.MultiError {
	return v.errs
}
