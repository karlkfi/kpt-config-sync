package parse

import (
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/parse/kptfile"
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

// AsResourceGroup accepts a list of core.Object and returns a list of FileObject
// with the following transformation:
// If no Kptfile is found, return the original FileObjects;
// If only one Kptfile is found, remove the Kptfile from the list and append a ResourceGroup;
// If multiple Kptfiles are found, return a MultipleKptfilesError
func AsResourceGroup(objs []core.Object) ([]core.Object, error) {
	kptfiles, resources := peelOffKptFiles(objs)
	switch len(kptfiles) {
	case 0:
		return objs, nil
	case 1:
		kpt, err := fromKptfile(kptfiles[0])
		if err != nil {
			return nil, err
		}
		err = validateKptfile(kpt)
		if err != nil {
			return nil, err
		}
		rg := kptfile.ResourceGroupFromKptFile(kpt, getIDs(resources))
		return append(resources, rg), nil
	default:
		return resources, MultipleKptfilesError(kptfiles...)
	}
}

func peelOffKptFiles(objs []core.Object) ([]core.Object, []core.Object) {
	var kptfiles []core.Object
	var resources []core.Object
	for _, obj := range objs {
		if isKptfile(obj) {
			kptfiles = append(kptfiles, obj)
		} else {
			resources = append(resources, obj)
		}
	}
	return kptfiles, resources
}

func getIDs(objs []core.Object) []kptfile.ObjMetadata {
	var ids []kptfile.ObjMetadata
	for _, obj := range objs {
		coreID := core.IDOf(obj)
		ids = append(ids, kptfile.ObjMetadata{
			Name:      coreID.Name,
			Namespace: coreID.Namespace,
			Group:     coreID.Group,
			Kind:      coreID.Kind,
		})
	}
	return ids
}

// MultipleKptfilesErrorCode is the error code for MultipleKptfilesError
const MultipleKptfilesErrorCode = "1059"

var multipleKptfilesError = status.NewErrorBuilder(MultipleKptfilesErrorCode)

// MultipleKptfilesError reports that there are multiple Kptfiles in a repo.
func MultipleKptfilesError(kptfiles ...core.Object) status.Error {
	resources := make([]id.Resource, len(kptfiles))
	for i, o := range kptfiles {
		resources[i] = o
	}
	return multipleKptfilesError.
		Sprintf("Namespace Repos may contain at most one Kptfile").
		BuildWithResources(resources...)
}

// InvalidKptfileErrorCode is the error code for an invalid Kptfile.
const InvalidKptfileErrorCode = "1062"

var invalidKptfileError = status.NewErrorBuilder(InvalidKptfileErrorCode)

// InvalidKptfileError reports that there is an invalid Inventory inside a Kptfile.
func InvalidKptfileError(s string, kptfiles core.Object) status.Error {
	return invalidKptfileError.
		Sprintf("Invalid inventory %s", s).
		BuildWithResources(kptfiles)
}

// isKptfile returns true if the object is a Kptfile.
func isKptfile(id core.Object) bool {
	return id.GroupVersionKind().GroupKind() == kinds.KptFile().GroupKind()
}

// fromKptfile converts the core.Object to a *Kptfile.
func fromKptfile(obj core.Object) (*kptfile.Kptfile, error) {
	if !isKptfile(obj) {
		return nil, errors.Errorf("not a Kptfile: %v", core.IDOf(obj))
	}
	if result, isKptfile := obj.(*kptfile.Kptfile); isKptfile {
		return result, nil
	}
	data, err := yaml.Marshal(obj)
	if err != nil {
		return nil, err
	}
	result := &kptfile.Kptfile{}
	err = yaml.Unmarshal(data, result)
	return result, err
}

func validateKptfile(kf *kptfile.Kptfile) status.Error {
	if kf == nil {
		return InvalidKptfileError("Kptfile shouldn't be nil", kf)
	}
	if kf.Inventory.Namespace == "" {
		return InvalidKptfileError(".inventory.namespace shouldn't be empty", kf)
	}
	if kf.Inventory.Identifier == "" {
		return InvalidKptfileError(".inventory.identifier shouldn't be empty", kf)
	}
	return nil
}
