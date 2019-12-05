package fakeinformers

import (
	"time"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	corev1 "k8s.io/client-go/informers/core/v1"
	corefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

// NewResourceQuotaInformer creates a fake SharedIndexInformer, injecting 'content'
// into the backing store.  Later calls on the client will pretend as if those
// objects were already inserted.
func NewResourceQuotaInformer(content ...runtime.Object) corev1.ResourceQuotaInformer {
	fakeClientSet := corefake.NewSimpleClientset(content...)
	factory := informers.NewSharedInformerFactory(
		fakeClientSet, 1*time.Minute)
	result := factory.Core().V1().ResourceQuotas()
	result.Informer()
	factory.Start(nil)
	glog.Infof("cache sync...")
	if !cache.WaitForCacheSync(nil, result.Informer().HasSynced) {
		panic("timed out waiting for cache sync")
	}
	return result
}
