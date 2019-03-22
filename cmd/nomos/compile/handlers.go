package compile

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ResourceEmitter is an interface for emitting Resource structs to a destination such as stdout
// or the filesystem.
type ResourceEmitter interface {
	// Emit emits a list of Resources that corresponds to the entire set of resources.
	Emit(items []Resource) error
}

// FilesystemHandler emits resources to the filesystem.
type FilesystemHandler struct {
	// root is the directory that will contain a 'cluster' and a 'namespace' subdirectory where
	// yaml files will be deposited in the appropriate locations.
	root string

	// force will remove the target directory if it exists prior to outputting for view.
	force bool
}

// NewFilesystemHandler creates a new FilesystemHandler.
func NewFilesystemHandler(path string, force bool) *FilesystemHandler {
	return &FilesystemHandler{root: path, force: force}
}

// Emit implements ResourceEmitter.
func (h *FilesystemHandler) Emit(items []Resource) error {
	if err := h.begin(); err != nil {
		return err
	}

	for _, r := range items {
		if err := h.emit(&r); err != nil {
			return err
		}
	}

	fmt.Printf("View has been emitted to %s\n", h.root)
	return nil
}

// Begin implements ResourceEmitter.
func (h *FilesystemHandler) begin() error {
	info, err := os.Stat(h.root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if h.force {
		return os.RemoveAll(h.root)
	}

	if !info.IsDir() {
		return errors.Errorf("%s is not a directory", h.root)
	}

	walkFunc := func(path string, info os.FileInfo, err error) error {
		if path == h.root {
			return nil
		}
		return errors.Errorf("%s is not empty", h.root)
	}
	return filepath.Walk(h.root, walkFunc)
}

// Emit implements ResourceEmitter.
func (h *FilesystemHandler) emit(r *Resource) error {
	path := filepath.Join(h.root, r.Path)

	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return err
	}

	_, err := os.Stat(path)
	if err == nil {
		return errors.Errorf("cannot output yaml to %s, file already exists", path)
	}
	if !os.IsNotExist(err) {
		return errors.Wrapf(err, "error during stat on file %s", path)
	}

	y, err := r.ToYAML()
	if err != nil {
		return err
	}

	glog.Infof("writing: %s", path)
	return ioutil.WriteFile(path, []byte(y), 0640)
}

// StdoutHandler emits resources to stdout with yaml-style '---' separators.
type StdoutHandler struct {
}

// NewStdoutHandler creates a new StdoutHandler.
func NewStdoutHandler() *StdoutHandler {
	return &StdoutHandler{}
}

// Emit implements ResourceEmitter.
func (h *StdoutHandler) Emit(items []Resource) error {
	var needSeparator bool
	for _, r := range items {
		if needSeparator {
			fmt.Println("---")
		} else {
			needSeparator = true
		}
		y, err := r.ToYAML()
		if err != nil {
			return err
		}
		fmt.Print(y)
	}
	return nil
}

const (
	clusterDir   = "cluster"
	namespaceDir = "namespace"
)

// normalizeResources converts all resources in all policies into a slice of Resource
func normalizeResources(ap *namespaceconfig.AllPolicies) []Resource {
	var objs []Resource
	objs = append(objs, normalizeGenRes(clusterDir, "", ap.ClusterConfig.Spec.Resources)...)
	for namespace, pn := range ap.NamespaceConfigs {
		path := filepath.Join(namespaceDir, namespace)
		objs = append(objs, Resource{
			Path: filepath.Join(path, "namespace.yaml"),
			Obj: runtime.RawExtension{
				Object: &corev1.Namespace{
					TypeMeta: metav1.TypeMeta{
						APIVersion: kinds.Namespace().Version,
						Kind:       kinds.Namespace().Kind,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:        namespace,
						Labels:      pn.Labels,
						Annotations: pn.Annotations,
					},
				},
			},
		})
		objs = append(objs, normalizeGenRes(path, namespace, pn.Spec.Resources)...)
	}
	return objs
}

// normalizeGenRes converts a slice of v1.GenericResources into a slice of Resource
func normalizeGenRes(pathPrefix string, namespace string, allRes []v1.GenericResources) []Resource {
	var objs []Resource
	for _, genRes := range allRes {
		for _, ver := range genRes.Versions {
			for _, obj := range ver.Objects {
				metaObj := obj.Object.(metav1.Object)
				kind := strings.ToLower(obj.Object.GetObjectKind().GroupVersionKind().Kind)
				path := filepath.Join(pathPrefix, fmt.Sprintf("%s-%s.yaml", kind, metaObj.GetName()))
				metaObj.SetNamespace(namespace)
				objs = append(objs, Resource{Path: path, Obj: obj})
			}
		}
	}
	return objs
}
