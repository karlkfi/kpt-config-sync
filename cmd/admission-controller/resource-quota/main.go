/*
Copyright 2017 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// This package runs the hierarchical resource quota admission controller
package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/google/stolos/pkg/admission-controller"
	"k8s.io/client-go/rest"
	admissionv1alpha1 "k8s.io/api/admission/v1alpha1"
	policynodemeta "github.com/google/stolos/pkg/client/meta"
	policynodeversions "github.com/google/stolos/pkg/client/informers/externalversions"
	informerspolicynodev1 "github.com/google/stolos/pkg/client/informers/externalversions/k8us/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"

	informerscorev1 "k8s.io/client-go/informers/core/v1"
)


var (
	listenAddr = flag.String(
		"listen_hostport", ":8000", "The hostport to listen to.")
	certFile = flag.String(
		"cert_file", "server.crt", "The server certificate file.")
	serverKeyFile = flag.String(
		"server_key", "server.key", "The server private key file.")
	handlerUrlPath = flag.String(
		"handler_url_path", "/", "The default handler URL path.")
)

// handleFunc is a shorthand for a HTTP handler function.
type handlerFunc func(http.ResponseWriter, *http.Request)

func serve(controller admission_controller.Admitter) handlerFunc {
	return func (w http.ResponseWriter, r *http.Request) {
		var body []byte
		if r.Body != nil {
			if data, err := ioutil.ReadAll(r.Body); err == nil {
				body = data
			}
		}

		// verify the content type is accurate
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			glog.Errorf("contentType=%s, expect application/json", contentType)
			return
		}

		review := admissionv1alpha1.AdmissionReview{}
		if err := json.Unmarshal(body, &review); err != nil {
			glog.Error(err)
			return
		}

		reviewStatus := controller.Admit(review)
		ar := admissionv1alpha1.AdmissionReview{
			Status: *reviewStatus,
		}

		resp, err := json.Marshal(ar)
		if err != nil {
			glog.Error(err)
		}
		if _, err := w.Write(resp); err != nil {
			glog.Error(err)
		}
	}
}

// NoCache positively turns off page caching.
func NoCache(handler handlerFunc) handlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set(
			"Cache-Control",
			"no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		handler(w, req)
	}

}

// WithStrictTransport decorates handler to require strict transport security
// when serving HTTPS request.
func WithStrictTransport(handler handlerFunc) handlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Strict-Transport-Security",
			"max-age=86400; includeSubdomains")
		handler(w, req)
	}
}

// ServeFunc returns the serving function for this server.  Use for testing.
func ServeFunc(controller admission_controller.Admitter) handlerFunc {
	return WithStrictTransport(NoCache(serve(controller)))
}

// Server configures and runs a TLS-enabled server from passed-in flags using
// the supplied handler.
func Server(handler handlerFunc) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc(*handlerUrlPath, handler)

	cfg := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}
	return &http.Server{
		Addr:      *listenAddr,
		Handler:   mux,
		TLSConfig: cfg,
		TLSNextProto: make(map[string]func(
			*http.Server, *tls.Conn, http.Handler), 0),
	}
}

func setupPolicyNodeInformer(config *rest.Config) (informerspolicynodev1.PolicyNodeInformer, error) {
	policyNodeClient, err := policynodemeta.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	policyNodeFactory := policynodeversions.NewSharedInformerFactory(
		policyNodeClient.PolicyHierarchy(), time.Minute,
	)
	policyNodeInformer := policyNodeFactory.K8us().V1().PolicyNodes()
	policyNodeInformer.Informer()
	policyNodeFactory.Start(nil)

	return policyNodeInformer, nil
}

func setupResourceQuotaInformer(config *rest.Config) (informerscorev1.ResourceQuotaInformer, error) {
	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	k8sFactory := informers.NewSharedInformerFactory(k8sClient, time.Minute)
	resourceQuotaInformer := k8sFactory.Core().V1().ResourceQuotas()
	resourceQuotaInformer.Informer()
	k8sFactory.Start(nil)

	return resourceQuotaInformer, nil
}

func main() {
	flag.Parse()
	glog.Infof("Hierarchical Resource Quota Admission Controller starting up")

	config, err := rest.InClusterConfig()
	if err != nil {
		glog.Fatal("Failed to load in cluster config: ", err)
	}
	policyNodeInformer, err := setupPolicyNodeInformer(config)
	if err != nil {
		glog.Fatal("Failed setting up policyNode informer: ", err)
	}
	resourceQuotaInformer, err := setupResourceQuotaInformer(config)
	if err != nil {
		glog.Fatal("Failed setting up resourceQuota informer: ", err)
	}
	glog.Infof("Waiting for informers to sync...")
	if !cache.WaitForCacheSync(nil,	policyNodeInformer.Informer().HasSynced, resourceQuotaInformer.Informer().HasSynced) {
		glog.Fatal("Failure while waiting for informers to sync")
	}

	server := Server(ServeFunc(admission_controller.NewResourceQuotaAdmitter(policyNodeInformer, resourceQuotaInformer)))

	glog.Infof("Hierarchical Resource Quota Admission Controller listening at: %v", *listenAddr)
	err = server.ListenAndServeTLS(*certFile, *serverKeyFile)
	if err != nil {
		glog.Fatal("ListenAndServe error: ", err)
	}
}
