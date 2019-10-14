package clusterconfig

import (
	"fmt"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/decode"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type crdKey struct{}

// AddCRDInfo adds CRDInfo into ast.Root.Data.
func AddCRDInfo(r *ast.Root, crdInfo *CRDInfo) status.Error {
	var err status.Error
	r.Data, err = ast.Add(r.Data, crdKey{}, crdInfo)
	return err
}

// GetCRDInfo returns CRDInfo stored in ast.Root.Data.
func GetCRDInfo(r *ast.Root) (*CRDInfo, status.Error) {
	clusterConfigInfo, err := ast.Get(r.Data, crdKey{})
	if err != nil {
		return nil, err
	}
	return clusterConfigInfo.(*CRDInfo), err
}

// CRDInfo has information about the CRD ClusterConfig needed to
// validate Custom Resource changes are valid.
type CRDInfo struct {
	// map of CR GroupKinds to corresponding CRD being deleted in the latest commit.
	pendingDelete map[schema.GroupKind]*v1beta1.CustomResourceDefinition
}

// NewCRDInfo returns a new CRDInfo.
func NewCRDInfo(decoder decode.Decoder, clusterConfig *v1.ClusterConfig,
	repoCRDs []*v1beta1.CustomResourceDefinition) (*CRDInfo, status.Error) {
	crds, err := crds(decoder, clusterConfig)
	if err != nil {
		return nil, err
	}

	return &CRDInfo{
		pendingDelete: pendingDelete(crds, repoCRDs),
	}, nil
}

// CRDPendingRemoval returns the CRD that is going to be removed in the latest commit for the corresponding Custom
// Resource in the FileObject.
func (c *CRDInfo) CRDPendingRemoval(o ast.FileObject) (*v1beta1.CustomResourceDefinition, bool) {
	gk := o.GroupVersionKind().GroupKind()
	crd, ok := c.pendingDelete[gk]
	return crd, ok
}

// crds returns the names and CustomResourceDefinitions of the CRDs in ClusterConfig.
func crds(decoder decode.Decoder, clusterConfig *v1.ClusterConfig) (map[string]*v1beta1.
	CustomResourceDefinition, status.Error) {
	crds := make(map[string]*v1beta1.CustomResourceDefinition)
	if clusterConfig == nil {
		return crds, nil
	}

	gvkrs, err := decoder.DecodeResources(clusterConfig.Spec.Resources)
	if err != nil {
		return nil, status.APIServerWrapf(err, "could not deserialize CRD in %s", v1.CRDClusterConfigName)
	}

	for gvk, rs := range gvkrs {
		if gvk != kinds.CustomResourceDefinition() {
			return nil, status.APIServerWrapf(err, "%s contains non-CRD resources: %v", v1.CRDClusterConfigName, gvk)
		}
		for _, r := range rs {
			crd, err := UnstructuredToCRD(r)
			if err != nil {
				return nil, status.APIServerWrapf(err, "could not deserialize CRD in %s", v1.CRDClusterConfigName)
			}
			crds[crd.Name] = crd
		}
	}
	return crds, nil
}

// pendingDelete returns the GroupKinds of the Custom Resources for the CRDs pending deletion in the current commit.
func pendingDelete(crds map[string]*v1beta1.CustomResourceDefinition,
	repoCRDs []*v1beta1.CustomResourceDefinition) map[schema.GroupKind]*v1beta1.CustomResourceDefinition {
	crdsInRepo := make(map[string]bool)
	for _, crd := range repoCRDs {
		crdsInRepo[crd.GetName()] = true
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

// StubbedCRDInfo returns a new CRDInfo with internal fields set.
func StubbedCRDInfo(pendingDelete map[schema.GroupKind]*v1beta1.CustomResourceDefinition) *CRDInfo {
	return &CRDInfo{
		pendingDelete: pendingDelete,
	}
}

// AsCRD returns the typed version of the CustomResourceDefinition passed in.
func AsCRD(o runtime.Object) (*v1beta1.CustomResourceDefinition, error) {
	if crd, ok := o.(*v1beta1.CustomResourceDefinition); ok {
		return crd, nil
	}
	// TODO(131853779): Should be able to always parse the typed CRD.
	if crd, ok := o.(*unstructured.Unstructured); ok {
		return UnstructuredToCRD(crd)
	}
	return nil, fmt.Errorf("could not generate a CRD from %T: %#v", o, o)
}

// UnstructuredToCRD returns the typed version of the CustomResourceDefinition of the Unstructured object.
func UnstructuredToCRD(u *unstructured.Unstructured) (*v1beta1.CustomResourceDefinition, error) {
	name, _, nameErr := unstructured.NestedString(u.Object, "metadata", "name")
	if nameErr != nil {
		return nil, nameErr
	}
	group, _, groupErr := unstructured.NestedString(u.Object, "spec", "group")
	if groupErr != nil {
		return nil, groupErr
	}
	scope, _, scopeErr := unstructured.NestedString(u.Object, "spec", "scope")
	if scopeErr != nil {
		return nil, scopeErr
	}
	kind, _, kindErr := unstructured.NestedString(u.Object, "spec", "names", "kind")
	if kindErr != nil {
		return nil, kindErr
	}
	plural, _, pluralErr := unstructured.NestedString(u.Object, "spec", "names", "plural")
	if pluralErr != nil {
		return nil, pluralErr
	}
	singular, _, singularErr := unstructured.NestedString(u.Object, "spec", "names", "singular")
	if singularErr != nil {
		return nil, singularErr
	}
	shortNames, _, shortNamesErr := unstructured.NestedStringSlice(u.Object, "spec", "names", "shortNames")
	if shortNamesErr != nil {
		return nil, shortNamesErr
	}
	categories, _, categoriesErr := unstructured.NestedStringSlice(u.Object, "spec", "names", "categories")
	if categoriesErr != nil {
		return nil, categoriesErr
	}
	version, _, versionErr := unstructured.NestedString(u.Object, "spec", "version")
	if versionErr != nil {
		return nil, versionErr
	}
	versions, _, versionsErr := unstructured.NestedSlice(u.Object, "spec", "versions")
	if versionsErr != nil {
		return nil, versionsErr
	}

	crd := &v1beta1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			Kind:       kinds.CustomResourceDefinition().Kind,
			APIVersion: kinds.CustomResourceDefinition().GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group: group,
			Scope: v1beta1.ResourceScope(scope),
			Names: v1beta1.CustomResourceDefinitionNames{
				Kind:       kind,
				Plural:     plural,
				Singular:   singular,
				ShortNames: shortNames,
				Categories: categories,
			},
		},
	}

	if len(versions) == 0 {
		crd.Spec.Versions = append(crd.Spec.Versions, v1beta1.CustomResourceDefinitionVersion{
			Name:    version,
			Served:  true,
			Storage: true,
		})
	}

	for _, v := range versions {
		crdVersion, _ := v.(map[string]interface{})
		name, _, nameErr := unstructured.NestedString(crdVersion, "name")
		if nameErr != nil {
			return nil, nameErr
		}
		served, _, servedErr := unstructured.NestedBool(crdVersion, "served")
		if servedErr != nil {
			return nil, servedErr
		}
		storage, _, storageErr := unstructured.NestedBool(crdVersion, "storage")
		if storageErr != nil {
			return nil, storageErr
		}
		crd.Spec.Versions = append(crd.Spec.Versions, v1beta1.CustomResourceDefinitionVersion{
			Name:    name,
			Served:  served,
			Storage: storage,
		})
	}

	return crd, nil
}
