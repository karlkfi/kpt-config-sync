package fakeinformers

import (
	"time"

	"github.com/golang/glog"
	pn_fake "github.com/google/nomos/clientgen/apis/fake"
	"github.com/google/nomos/clientgen/informer"
	pn_v1 "github.com/google/nomos/clientgen/informer/policyhierarchy/v1"
	"github.com/google/nomos/pkg/syncer/parentindexer"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	core_v1 "k8s.io/client-go/informers/core/v1"
	core_fake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

// NewPolicyNodeInformer creates a fake SharedIndexInformer, injecting 'content'
// into the backing store.  Later calls on the client will pretend as if those
// objects were already inserted.
func NewPolicyNodeInformer(content ...runtime.Object) pn_v1.PolicyNodeInformer {
	fakeClientSet := pn_fake.NewSimpleClientset(content...)
	factory := informer.NewSharedInformerFactory(
		fakeClientSet, 1*time.Minute)
	informer := factory.Nomos().V1().PolicyNodes()
	if err := informer.Informer().AddIndexers(parentindexer.Indexer()); err != nil {
		panic(err)
	}
	factory.Start(nil)
	glog.Infof("cache sync...")
	if !cache.WaitForCacheSync(nil, informer.Informer().HasSynced) {
		panic("timed out waiting for cache sync")
	}
	return informer
}

// NewResourceQuotaInformer creates a fake SharedIndexInformer, injecting 'content'
// into the backing store.  Later calls on the client will pretend as if those
// objects were already inserted.
func NewResourceQuotaInformer(content ...runtime.Object) core_v1.ResourceQuotaInformer {
	fakeClientSet := core_fake.NewSimpleClientset(content...)
	factory := informers.NewSharedInformerFactory(
		fakeClientSet, 1*time.Minute)
	informer := factory.Core().V1().ResourceQuotas()
	informer.Informer()
	factory.Start(nil)
	glog.Infof("cache sync...")
	if !cache.WaitForCacheSync(nil, informer.Informer().HasSynced) {
		panic("timed out waiting for cache sync")
	}
	return informer
}
