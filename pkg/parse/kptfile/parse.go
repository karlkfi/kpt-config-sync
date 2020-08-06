package kptfile

import (
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/status"
)

// AsResourceGroup accepts a list of FileObject and returns a list of FileObject
// with the following transformation:
// If no Kptfile is found, return the original FileObjects;
// If only one Kptfile is found, remove the Kptfile from the list and append a ResourceGroup;
// If multiple Kptfiles are found, return a MultipleKptfilesError
func AsResourceGroup(objs []ast.FileObject) ([]ast.FileObject, error) {
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
		paths := make([]string, len(kptfiles))
		for i, kptfile := range kptfiles {
			paths[i] = kptfile.OSPath()
		}
		return resources, status.MultipleKptfilesError(paths...)
	}
}

func peelOffKptFiles(objs []ast.FileObject) ([]ast.FileObject, []ast.FileObject) {
	var kptfiles []ast.FileObject
	var resources []ast.FileObject
	for _, obj := range objs {
		if isKptfile(core.IDOf(obj)) {
			kptfiles = append(kptfiles, obj)
		} else {
			resources = append(resources, obj)
		}
	}
	return kptfiles, resources
}

func getIDs(objs []ast.FileObject) []ObjMetadata {
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
