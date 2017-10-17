package authorizer

import (
	"time"

	"github.com/golang/glog"
	"github.com/google/stolos/pkg/client/informers/externalversions"
	"github.com/google/stolos/pkg/client/policyhierarchy/fake"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

// NewTestInformer creates a fake SharedIndexInformer, injecting 'content'
// into the backing store.  Later calls on the client will pretend as if those
// objects were already inserted.
func NewTestInformer(content ...runtime.Object) cache.SharedIndexInformer {
	fakeClientSet := fake.NewSimpleClientset(content...)
	factory := externalversions.NewSharedInformerFactory(
		fakeClientSet, 1*time.Minute)
	informer := factory.K8us().V1().PolicyNodes().Informer()
	factory.Start(nil)
	glog.V(1).Infof("cache sync...")
	if !cache.WaitForCacheSync(nil, informer.HasSynced) {
		panic("timed out waiting for cache sync")
	}
	return informer
}
