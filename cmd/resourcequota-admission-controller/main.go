package main

import (
	"flag"
	"net"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/cmd/resourcequota-admission-controller/controller"
	"github.com/google/nomos/pkg/admissioncontroller"
	"github.com/google/nomos/pkg/admissioncontroller/resourcequota"
	"github.com/google/nomos/pkg/service"
	"github.com/google/nomos/pkg/util/log"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

var (
	caBundleFile = flag.String("ca-cert", "ca.crt", "Webhook server bundle cert used by api-server to authenticate the webhook server.")
	enablemTLS   = flag.Bool("enable-mutual-tls", false, "If set, enables mTLS verification of the client connecting to the admission controller.")
	resyncPeriod = flag.Duration("resync-period", time.Minute, "The resync period for the admission controller.")
)

func main() {
	flag.Parse()
	log.Setup()

	glog.Info("Hierarchical Resource Quota Admission Controller starting up")

	config, err := rest.InClusterConfig()
	if err != nil {
		glog.Fatalf("Failed to load in cluster config: %+v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatalf("Failed to create client set: %+v", err)
	}
	var clientCert []byte
	if *enablemTLS {
		clientCert, err = admissioncontroller.GetAPIServerCert(clientset)
		if err != nil {
			glog.Fatalf("Failed to get client cert: %+v", err)
		}
	}
	hierarchicalQuotaInformer, err := admissioncontroller.SetupHierarchicalQuotaInformer(config, *resyncPeriod)
	if err != nil {
		glog.Fatalf("Failed setting up hierarchicalQuota informer: %+v", err)
	}
	resourceQuotaInformer, err := controller.SetupResourceQuotaInformer(config)
	if err != nil {
		glog.Fatalf("Failed setting up resourceQuota informer: %+v", err)
	}

	glog.Info("Waiting for informers to sync...")
	if !cache.WaitForCacheSync(nil, hierarchicalQuotaInformer.Informer().HasSynced, resourceQuotaInformer.Informer().HasSynced) {
		glog.Fatal("Failure while waiting for informers to sync")
	}

	go service.ServeMetrics()

	server := service.Server(
		admissioncontroller.ServeFunc(
			resourcequota.NewAdmitter(resourceQuotaInformer, hierarchicalQuotaInformer)),
		clientCert)

	stopChannel := make(chan struct{})
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		glog.Fatalf("Failed to start https listener: %+v", err)
	}
	// nolint: errcheck
	defer listener.Close()

	go admissioncontroller.ServeTLS(server, listener, stopChannel)

	// Wait for endpoint to come up before self-registering
	err = admissioncontroller.WaitForEndpoint(
		clientset, controller.ControllerName, controller.ControllerNamespace, admissioncontroller.EndpointRegistrationTimeout)
	if err != nil {
		glog.Fatalf("Failed waiting for endpoint: %+v", err)
	}

	// Finally register the webhook to block admission according to quota policy
	if err := controller.SelfRegister(clientset, *caBundleFile); err != nil {
		glog.Fatalf("Failed to register webhook: %+v", err)
	}

	<-stopChannel
}
