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
	log "github.com/golang/glog"
	"io/ioutil"
	authz "k8s.io/api/authorization/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
)

var (
	listenAddr = flag.String(
		"listen_hostport", ":443", "The hostport to listen to.")
	certFile = flag.String(
		"cert_file", "server.crt", "The server certificate file")
	serverKeyFile = flag.String(
		"server_key", "server.key", "The server key file.")
	handlerUrlPath = flag.String(
		"handler_url_path", "/authorize", "The default handler URL path.")
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
		log.Infof("Method: %v, URL: %v", req.Method, req.URL)
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
func Responder(writer http.ResponseWriter, req *http.Request) {
	var body []byte
	if req.Body != nil {
		if data, err := ioutil.ReadAll(req.Body); err == nil {
			body = data
		}
	}
	contentType := req.Header.Get("Content-Type")
	if contentType != "application/json" {
		log.Errorf(
			"contentType='%v', expect application/json", contentType)
		writer.WriteHeader(http.StatusUnsupportedMediaType)
		return
	}

	var reviewRequest authz.SubjectAccessReview
	err := json.Unmarshal(body, &reviewRequest)
	if err != nil {
		log.Errorf("Could not unmarshal as request spec: %v", body)
		return
	}
	// TODO(filmil): Check the request sanity

	reviewResponse := authz.SubjectAccessReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SubjectAccessReview",
			APIVersion: "authorization.k8s.io/v1beta1",
		},
		Status: authz.SubjectAccessReviewStatus{
			Allowed: true,
		},
	}
	resp, err := json.Marshal(reviewResponse)
	if err != nil {
		log.Errorf("While marshalling response: %v", err)
		return
	}
	writer.Write(resp)
}

// ServeFunc returns the serving function for this server.  Use for testing.
func ServeFunc() handlerFunc {
	return WithStrictTransport(WithRequestLogging(NoCache(Responder)))
}

// Server configures and runs a TLS-enabled server from passed-in flags using
// the supplied handler.
func Server(handler handlerFunc) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc(*handlerUrlPath, handler)
	// TODO(filmil): Check how to install a handler that returns 404 for
	// everything else.

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

func main() {
	flag.Parse()
	srv := Server(ServeFunc())
	log.Infof("Listening at: %v", *listenAddr)
	err := srv.ListenAndServeTLS(*certFile, *serverKeyFile)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
