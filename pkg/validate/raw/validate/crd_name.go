package validate

import (
	"fmt"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/clusterconfig"
)

// CRDName returns an error if the CRD's name does not match the Kubernetes
// specification:
// https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/
func CRDName(obj ast.FileObject) status.Error {
	if obj.GroupVersionKind().GroupKind() != kinds.CustomResourceDefinition() {
		return nil
	}

	crd, err := clusterconfig.AsCRD(obj.Object)
	if err != nil {
		return err
	}
	expectedName := fmt.Sprintf("%s.%s", crd.Spec.Names.Plural, crd.Spec.Group)
	if crd.Name != expectedName {
		return nonhierarchical.InvalidCRDNameError(obj, expectedName)
	}

	return nil
}
