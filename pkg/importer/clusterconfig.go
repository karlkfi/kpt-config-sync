package importer

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type clusterConfigKey struct{}

// AddCRDClusterConfigInfo adds CRDClusterConfigInfo into ast.Root.Data.
func AddCRDClusterConfigInfo(r *ast.Root, crdInfo *CRDClusterConfigInfo) status.Error {
	var err status.Error
	r.Data, err = ast.Add(r.Data, clusterConfigKey{}, crdInfo)
	return err
}

// GetCRDClusterConfigInfo returns CRDClusterConfigInfo stored in ast.Root.Data.
func GetCRDClusterConfigInfo(r *ast.Root) (*CRDClusterConfigInfo, status.Error) {
	clusterConfigInfo, err := ast.Get(r.Data, clusterConfigKey{})
	if err != nil {
		return nil, err
	}
	return clusterConfigInfo.(*CRDClusterConfigInfo), err
}

// CRDClusterConfigInfo has information about the CRD ClusterConfig needed to
// validate Custom Resource changes are valid.
type CRDClusterConfigInfo struct {
	// map of CR GroupKinds to corresponding CRD being deleted in the latest commit.
	pendingDelete map[schema.GroupKind]*v1beta1.CustomResourceDefinition
}

// NewCRDClusterConfigInfo returns a new CRDClusterConfigInfo.
func NewCRDClusterConfigInfo(clusterConfig *v1.ClusterConfig, os []*ast.ClusterObject) *CRDClusterConfigInfo {
	return &CRDClusterConfigInfo{
		pendingDelete: pendingDelete(crds(clusterConfig), os),
	}
}

// CRDPendingRemoval returns the CRD that is going to be removed in the latest commit for the corresponding Custom
// Resource in the FileObject.
func (c *CRDClusterConfigInfo) CRDPendingRemoval(o ast.FileObject) (*v1beta1.CustomResourceDefinition, bool) {
	gk := o.GroupVersionKind().GroupKind()
	crd, ok := c.pendingDelete[gk]
	return crd, ok
}

// crds returns the names and CustomResourceDefinitions of the CRDs in ClusterConfig.
func crds(clusterConfig *v1.ClusterConfig) map[string]*v1beta1.CustomResourceDefinition {
	crds := make(map[string]*v1beta1.CustomResourceDefinition)
	if clusterConfig == nil {
		return crds
	}

	for _, r := range clusterConfig.Spec.Resources {
		for _, v := range r.Versions {
			for _, obj := range v.Objects {
				crd := obj.Object.(*v1beta1.CustomResourceDefinition)
				crds[crd.Name] = crd
			}
		}
	}
	return crds
}

// pendingDelete returns the GroupKinds of the Custom Resources for the CRDs pending deletion in the current commit.
func pendingDelete(crds map[string]*v1beta1.CustomResourceDefinition,
	os []*ast.ClusterObject) map[schema.GroupKind]*v1beta1.CustomResourceDefinition {
	crdsInRepo := make(map[string]bool)
	for _, o := range os {
		if o.GroupVersionKind() != kinds.CustomResourceDefinition() {
			continue
		}
		crdsInRepo[o.Name()] = true
	}

	pendingDelete := make(map[schema.GroupKind]*v1beta1.CustomResourceDefinition)
	for name, crd := range crds {
		if !crdsInRepo[name] {
			gk := schema.GroupKind{Group: crd.Spec.Group, Kind: crd.Spec.Names.Kind}
			pendingDelete[gk] = crd
		}
	}
	return pendingDelete
}

// StubbedCRDClusterConfigInfo returns a new CRDClusterConfigInfo with internal fields set.
func StubbedCRDClusterConfigInfo(pendingDelete map[schema.GroupKind]*v1beta1.CustomResourceDefinition) *CRDClusterConfigInfo {
	return &CRDClusterConfigInfo{
		pendingDelete: pendingDelete,
	}
}
