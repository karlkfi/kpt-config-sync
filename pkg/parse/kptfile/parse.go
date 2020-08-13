package kptfile

import (
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/status"
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
		kpt, err := kptfileFrom(kptfiles[0])
		if err != nil {
			return resources, err
		}
		rg := resourceGroupFromKptFile(kpt, getIDs(resources))
		return append(resources, rg), nil
	default:
		names := make([]string, len(kptfiles))
		for i, kptfile := range kptfiles {
			names[i] = kptfile.GetName()
		}
		return resources, status.MultipleKptfilesError(names...)
	}
}

func peelOffKptFiles(objs []core.Object) ([]core.Object, []core.Object) {
	var kptfiles []core.Object
	var resources []core.Object
	for _, obj := range objs {
		if isKptfile(core.IDOf(obj)) {
			kptfiles = append(kptfiles, obj)
		} else {
			resources = append(resources, obj)
		}
	}
	return kptfiles, resources
}

func getIDs(objs []core.Object) []ObjMetadata {
	var ids []ObjMetadata
	for _, obj := range objs {
		coreID := core.IDOf(obj)
		ids = append(ids, ObjMetadata{
			Name:      coreID.Name,
			Namespace: coreID.Namespace,
			Group:     coreID.Group,
			Kind:      coreID.Kind,
		})
	}
	return ids
}
