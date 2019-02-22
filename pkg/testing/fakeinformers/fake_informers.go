package fakeinformers

import (
	"time"

	"github.com/golang/glog"
	pnfake "github.com/google/nomos/clientgen/apis/fake"
	"github.com/google/nomos/clientgen/informer"
	informersv1 "github.com/google/nomos/clientgen/informer/policyhierarchy/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	corev1 "k8s.io/client-go/informers/core/v1"
	corefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

// NewHierarchicalQuotaInformer creates a fake SharedIndexInformer, injecting 'content'
// into the backing store.  Later calls on the client will pretend as if those
// objects were already inserted.
func NewHierarchicalQuotaInformer(content ...runtime.Object) informersv1.HierarchicalQuotaInformer {
	fakeClientSet := pnfake.NewSimpleClientset(content...)
	factory := informer.NewSharedInformerFactory(
		fakeClientSet, 1*time.Minute)
	informer := factory.Nomos().V1().HierarchicalQuotas()
	informer.Informer()
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
func NewResourceQuotaInformer(content ...runtime.Object) corev1.ResourceQuotaInformer {
	fakeClientSet := corefake.NewSimpleClientset(content...)
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
