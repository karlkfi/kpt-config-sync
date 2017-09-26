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

// Scaffolding of a Kubernetes webhook authorizer.
package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"github.com/coreos/go-systemd/daemon"
	"github.com/golang/glog"
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/authorizer"
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/client/policyhierarchy"
	apierrors "github.com/pkg/errors"
	"io/ioutil"
	authz "k8s.io/api/authorization/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"net/http"
)

var (
	listenAddr = flag.String(
		"listen_hostport", ":8443", "The hostport to listen to.")
	caCertFile = flag.String(
		"ca_cert_file", "ca.crt", "The Certificate Authority certificate used to verify the identity of the remote end at TLS handshake time.")
	certFile = flag.String(
		"cert_file", "server.crt", "The server certificate file.")
	serverKeyFile = flag.String(
		"server_key", "server.key", "The server private key file.")
	handlerUrlPath = flag.String(
		"handler_url_path", "/authorize", "The default handler URL path.")
	notifySystemd = flag.Bool(
		"notify_systemd", false, "Whether to notify systemd that the daemon is ready to serve.")

	// Normally, outside of a cluster, we would get this information from
	// the cluster configuration.  Inside a cluster, for example within a pod,
	// this information we'd get from the default "in-cluster" client.  But
	// for a program that runs on the master, we must assemble the required
	// information directly.
	apiserverAddr = flag.String(
		"apiserver_hostport", "localhost:8443", "The hostport address of the Kubernetes API server")
)

// handleFunc is a shorthand for a HTTP handler function.
type handlerFunc func(http.ResponseWriter, *http.Request)

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

// WithRequestLogging decorates handler with a log statement that prints the
// method and the URL requested.
func WithRequestLogging(handler handlerFunc) handlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		glog.Infof("Method: %v, URL: %v", req.Method, req.URL)
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

// Responder writes a basic message out.
// See "Request Payloads" at:
// https://kubernetes-v1-4.github.io/docs/admin/authorization for details.
func Responder(a *authorizer.Authorizer) handlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		var body []byte
		if req.Body != nil {
			if data, err := ioutil.ReadAll(req.Body); err == nil {
				body = data
			}
		}

		// Run the type sanity checks: expect the correct content type,
		// then try to deserialize, and bomb out on invalid TypeMeta.
		contentType := req.Header.Get("Content-Type")
		if contentType != "application/json" {
			glog.Errorf(
				"contentType='%v', expect application/json",
				contentType)
			writer.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}
		var reviewRequest authz.SubjectAccessReview
		err := json.Unmarshal(body, &reviewRequest)
		if err != nil {
			glog.Errorf(
				"Could not unmarshal as request spec: %v",
				body)
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		// TODO(filmil): Is there a "canonical" way to check this?
		if reviewRequest.TypeMeta != authorizer.TypeMeta {
			glog.Errorf("Invalid TypeMeta: %v",
				reviewRequest.TypeMeta)
			writer.WriteHeader(http.StatusBadRequest)
			return
		}

		// Process the request: call authorizer here.
		glog.V(1).Infof("Request: %+v", string(body))
		resp, err := json.Marshal(authz.SubjectAccessReview{
			TypeMeta: authorizer.TypeMeta,
			Status:   *a.Authorize(&reviewRequest.Spec),
		})
		if err != nil {
			glog.Errorf("While marshalling response: %v", err)
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}

		// If we survived up to here, write the response out and done.
		glog.V(2).Infof("Response: %+v", string(resp))
		writer.Write(resp)
	}
}

// ServeFunc returns the serving function for this server.  Use for testing.
func ServeFunc(a *authorizer.Authorizer) handlerFunc {
	return WithStrictTransport(
		WithRequestLogging(
			NoCache(
				Responder(a))))
}

// Server configures and runs a TLS-enabled server from passed-in flags using
// the supplied handler.
func Server(handler handlerFunc) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc(*handlerUrlPath, handler)
	// TODO(filmil): Check how to install a handler that returns 404 for
	// everything else.

	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{
			tls.CurveP521,
			tls.CurveP384,
			tls.CurveP256},
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

// LogApiVersion logs the API server version before proceeding.
func LogApiVersion(kubernetesConfig *rest.Config) {
	clientSet, err := kubernetes.NewForConfig(kubernetesConfig)
	if err != nil {
		glog.Error(apierrors.Wrapf(err,
			"Could not contact the Kubernetes API server at: %v",
			*apiserverAddr))
		return
	}
	_, err = clientSet.CoreV1().Pods("default").Get("", metav1.GetOptions{})
	if errors.IsNotFound(err) {
		glog.Error(apierrors.Wrapf(err, "Pod not found."))
		return
	}
	if statusError, isStatus := err.(*errors.StatusError); isStatus {
		glog.Error(apierrors.Wrapf(statusError, "Error getting pod"))
	}

	glog.Infof("Found some pods.")
}

// newKubernetesClientConfig creates a configuration for the client-go code.
//
// From outside a cluster, this information comes from the cluster
// configuration file.  From inside the cluster, this information comes
// from the "in-cluster" Kubernetes client.  For a daemon running on
// the master, we have to assemble this information from the information set
// given to us in flags.
func newKubernetesClientConfig() *rest.Config {
	return &rest.Config{
		Host: *apiserverAddr,
		TLSClientConfig: rest.TLSClientConfig{
			CertFile: *certFile,
			KeyFile:  *serverKeyFile,
			CAFile:   *caCertFile,
		},
	}
}

func main() {
	flag.Parse()
	glog.Infof("Webhook authorizer listening at: %v", *listenAddr)
	glog.Infof("Using server certificate file: %v", *certFile)
	glog.Infof("Using server private key file: %v", *serverKeyFile)
	glog.Infof("Using CA file: %v", *caCertFile)

	clientConfig := newKubernetesClientConfig()
	policyHierarchyClient, err := policyhierarchy.NewForConfig(clientConfig)
	if err != nil {
		glog.Fatalf("Could not create Kubernetes API client: %v", err)
	}

	srv := Server(ServeFunc(authorizer.New(policyHierarchyClient.K8usV1())))

	// Demo the connection to the apiserver.
	go LogApiVersion(clientConfig)

	// Notifies the monitor daemon that we're ready to start serving.
	// But only if the daemon is actually on the other side, since
	// the notification writes into a Unix socket under the hood.
	if *notifySystemd {
		daemon.SdNotify( /*unsetEnvironment=*/ false, "READY=1")
	}

	err = srv.ListenAndServeTLS(*certFile, *serverKeyFile)
	if err != nil {
		glog.Fatal("ListenAndServe: ", err)
	}
}
