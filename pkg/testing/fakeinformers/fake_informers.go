package fakeinformers

import (
	"time"

	"github.com/golang/glog"
	"github.com/google/stolos/pkg/client/informers/externalversions"
	pn_v1 "github.com/google/stolos/pkg/client/informers/externalversions/k8us/v1"
	pn_fake "github.com/google/stolos/pkg/client/policyhierarchy/fake"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	core_fake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/informers"
	core_v1 "k8s.io/client-go/informers/core/v1"
)

// NewTestInformer creates a fake SharedIndexInformer, injecting 'content'
// into the backing store.  Later calls on the client will pretend as if those
// objects were already inserted.
func NewPolicyNodeInformer(content ...runtime.Object) pn_v1.PolicyNodeInformer {
	fakeClientSet := pn_fake.NewSimpleClientset(content...)
	factory := externalversions.NewSharedInformerFactory(
		fakeClientSet, 1*time.Minute)
	informer := factory.K8us().V1().PolicyNodes()
	informer.Informer()
	factory.Start(nil)
	glog.Infof("cache sync...")
	if !cache.WaitForCacheSync(nil, informer.Informer().HasSynced) {
		panic("timed out waiting for cache sync")
	}
	return informer
}


// NewTestInformer creates a fake SharedIndexInformer, injecting 'content'
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
