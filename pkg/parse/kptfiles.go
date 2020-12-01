package parse

import (
	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/parse/resourcegroup"
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
		if kpt != nil {
			rg := resourcegroup.FromKptFile(kpt, getIDs(resources))
			resources = append(resources, rg)
		}
		return resources, nil
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

func getIDs(objs []core.Object) []resourcegroup.ObjMetadata {
	var ids []resourcegroup.ObjMetadata
	for _, obj := range objs {
		coreID := core.IDOf(obj)
		ids = append(ids, resourcegroup.ObjMetadata{
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
func fromKptfile(obj core.Object) (*kptfile.KptFile, error) {
	result, err := toKptfile(obj)
	if err != nil {
		return nil, err
	}
	// Skip the Kptfile when it doesn't specify the Inventory field.
	if isEmptyInventory(result.Inventory) {
		return nil, nil
	}
	err = validateInventory(result.Inventory, obj)
	return result, err
}

func toKptfile(obj core.Object) (*kptfile.KptFile, error) {
	if !isKptfile(obj) {
		return nil, errors.Errorf("not a Kptfile: %v", core.IDOf(obj))
	}
	data, err := yaml.Marshal(obj)
	if err != nil {
		return nil, err
	}
	result := &kptfile.KptFile{}
	err = yaml.Unmarshal(data, result)
	return result, err
}

func validateInventory(inv kptfile.Inventory, kfObj core.Object) error {
	if inv.Namespace == "" {
		return InvalidKptfileError(".inventory.namespace shouldn't be empty", kfObj)
	}
	if inv.Name == "" {
		return InvalidKptfileError(".inventory.name shouldn't be empty", kfObj)
	}
	return nil
}

func isEmptyInventory(inv kptfile.Inventory) bool {
	if inv.Namespace != "" {
		return false
	}
	if inv.Name != "" {
		return false
	}
	if inv.InventoryID != "" {
		return false
	}
	if len(inv.Labels) > 0 {
		return false
	}
	if len(inv.Annotations) > 0 {
		return false
	}
	return true
}
