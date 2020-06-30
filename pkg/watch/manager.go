package watch

import (
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/parse/declaredresources"
)

// defaultResyncTime has the same value as in controller-runtime
var defaultResyncTime = 10 * time.Hour

// Manager accepts new resource lists that are parsed from Git and then
// updates declared resources and get GVKs.
type Manager struct {
	// cfg is the rest config used to talk to apiserver.
	cfg *rest.Config

	// mapper is the RESTMapper to use for mapping GroupVersionKinds to Resources.
	mapper meta.RESTMapper

	// resources is the declared resources that parsed from Git.
	resources *declaredresources.DeclaredResources

	// Informers maps GVKs to their associated informers.
	informers map[schema.GroupVersionKind]mapEntry

	// resync is the time that the informers are resynced.
	resync time.Duration

	// createInformerFunc is the function to create an Informer.
	createInformerFunc createInformerFunc
}

type mapEntry struct {
	informer cache.SharedIndexInformer

	// stopCh is the stop channel associated with the informer
	stopCh chan struct{}
}

// Options contains options for creating a watch manager.
type Options struct {
	Mapper meta.RESTMapper
	// Resync is the frequency that informers are resynced.
	// It will list all resources and rehydrate the informer's store.
	Resync       time.Duration
	InformerFunc createInformerFunc
}

// DefaultOptions return the default options:
// - set up resync time to 1 hour
// - create discovery RESTmapper from the passed rest.Config
// - use createInformer to create informers
func DefaultOptions(cfg *rest.Config) (*Options, error) {
	mapper, err := apiutil.NewDiscoveryRESTMapper(cfg)
	if err != nil {
		return nil, err
	}

	return &Options{
		Mapper:       mapper,
		Resync:       defaultResyncTime,
		InformerFunc: createInformer,
	}, nil
}

// NewManager starts a new watch manager
func NewManager(cfg *rest.Config, options *Options) (*Manager, error) {
	if options == nil {
		var err error
		options, err = DefaultOptions(cfg)
		if err != nil {
			return nil, err
		}
	}

	return &Manager{
		cfg:                cfg,
		resources:          declaredresources.NewDeclaredResources(),
		informers:          make(map[schema.GroupVersionKind]mapEntry),
		resync:             options.Resync,
		createInformerFunc: options.InformerFunc,
		mapper:             options.Mapper,
	}, nil
}

// Update accepts new resource list and takes following actions:
// - start informers for any GroupKind that is present in the new
//   resource list, but weren't present in previous resource list.
// - stop informers for any GroupKind that is not present
//   in the new resource list
func (m *Manager) Update(objects []core.Object) error {
	err := m.resources.UpdateDecls(objects)
	if err != nil {
		return err
	}
	gkSet := m.resources.GetGKSet()

	// Stop obsolete Informers
	for gvk := range m.informers {
		if _, found := gkSet[gvk.GroupKind()]; !found {
			m.stopInformer(gvk)
		} else {
			// We can remove this GK from gkSet because we know
			// it is present in informers so we won't need to start a new informer for it below.
			delete(gkSet, gvk.GroupKind())
		}
	}

	// Start new Informers
	for gk, version := range gkSet {
		gvk := gk.WithVersion(version)
		if _, found := m.informers[gvk]; !found {
			if err := m.startInformer(gvk); err != nil {
				return err
			}
		}
	}
	return nil
}

// startInformer starts a shared informer for a GVK
func (m *Manager) startInformer(gvk schema.GroupVersionKind) error {
	_, found := m.informers[gvk]
	if found {
		// The informer is already started
		return nil
	}
	entry, err := m.createInformerFunc(gvk, m.mapper, m.cfg, m.resync)
	if err != nil {
		return err
	}
	m.informers[gvk] = entry
	return nil
}

// stopInformer stops a shared informer for a GVK
func (m *Manager) stopInformer(gvk schema.GroupVersionKind) {
	entry, found := m.informers[gvk]
	if !found {
		// The informer is already stopped
		return
	}

	// The informer will be stopped when stopCh is closed.
	close(entry.stopCh)
	delete(m.informers, gvk)
}
